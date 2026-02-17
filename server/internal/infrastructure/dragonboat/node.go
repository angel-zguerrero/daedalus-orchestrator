package dragonboat

import (
	"bytes"
	"context"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"encoding/gob"
	"errors"
	"math"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/client"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/statemachine"
	"github.com/rs/zerolog/log"

	"deadalus-orch/server/internal/infrastructure/db"
	appConfig "deadalus-orch/server/internal/pkg/config"
)

// RaftNode represents a single node (replica) participating in a Dragonboat Raft consensus group (shard).
// It encapsulates the Dragonboat NodeHost, configuration for the replica, and methods to interact with it.
type RaftNode struct {
	NH             *dragonboat.NodeHost                                                   // The underlying Dragonboat NodeHost instance.
	ShardID        uint64                                                                 // The ID of the shard (Raft consensus group) this node belongs to.
	ReplicaID      uint64                                                                 // The unique ID of this replica within the shard.
	SelfMember     Member                                                                 // Details of the current node (e.g., address, port).
	InitialMembers []Member                                                               // List of initial members for bootstrapping a new shard.
	Join           bool                                                                   // Flag indicating if this node is joining an existing shard.
	Roles          []NodeRole                                                             // Roles assigned to this node (e.g., consensus, scheduler).
	stateMachine   func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine // A factory function that creates an instance of the on-disk state machine for this replica.
	mu             sync.RWMutex
	stopped        bool
	readSem        chan struct{} // max 100 inflight reads
}

type RaftTuningParams struct {
	ElectionRTT        int
	HeartbeatRTT       int
	SnapshotEntries    uint64
	CompactionOverhead uint64
}

// GetClusterEnvironmentType determines if we're running in a small distributed cluster
// and returns optimized parameters accordingly
func getClusterEnvironmentMultiplier() (heartbeatMultiplier, electionMultiplier float64) {
	// For small clusters, we still want to be slightly conservative but not as much as before
	if appConfig.GlobalConfiguration.MaxShards <= 20 {
		return 1.1, 1.2 // 10% higher heartbeat, 20% higher election RTTs
	} else if appConfig.GlobalConfiguration.MaxShards <= 100 {
		return 1.05, 1.1 // 5% higher heartbeat, 10% higher election RTTs
	}
	return 1.0, 1.0 // Standard values for larger clusters
}

func RecommendRaftParamsForShards() RaftTuningParams {
	numShards := appConfig.GlobalConfiguration.MaxShards

	// Get environment-specific multipliers for small distributed clusters
	heartbeatMultiplier, electionMultiplier := getClusterEnvironmentMultiplier()

	// Base values optimized for distributed 3-node clusters
	// HeartbeatRTT should be higher for network stability
	var baseHeartbeatRTT int
	switch {
	case numShards <= 100:
		baseHeartbeatRTT = 2 // Increased from 1 for better stability
	case numShards <= 200:
		baseHeartbeatRTT = 3
	case numShards <= 500:
		baseHeartbeatRTT = 5
	default:
		baseHeartbeatRTT = 8
	}

	var baseElectionRTT int
	// ElectionRTT should grow as shard count increases, to avoid split votes.
	if numShards <= 100 {
		baseElectionRTT = 20 // Increased from 10 for better stability
	} else if numShards <= 300 {
		baseElectionRTT = 30
	} else if numShards <= 600 {
		baseElectionRTT = 50
	} else if numShards <= 1000 {
		baseElectionRTT = 80
	} else {
		baseElectionRTT = 100
	}

	// Apply environment-specific multipliers
	heartbeatRTT := int(float64(baseHeartbeatRTT) * heartbeatMultiplier)
	electionRTT := int(float64(baseElectionRTT) * electionMultiplier)

	// SnapshotEntries determines how often full snapshots are taken.
	// Higher shard count = less frequent snapshots to reduce IO pressure.
	var snapshotEntries uint64
	switch {
	case numShards <= 100:
		snapshotEntries = 2000
	case numShards <= 300:
		snapshotEntries = 5000
	case numShards <= 600:
		snapshotEntries = 10000
	case numShards <= 1000:
		snapshotEntries = 20000
	default:
		snapshotEntries = 40000
	}

	// CompactionOverhead is how many extra entries to retain after snapshot.
	// Higher values help avoid replays from scratch during restore.
	compactionOverhead := uint64(math.Min(512, float64(snapshotEntries/4)))

	return RaftTuningParams{
		ElectionRTT:        electionRTT,
		HeartbeatRTT:       heartbeatRTT,
		SnapshotEntries:    snapshotEntries,
		CompactionOverhead: compactionOverhead,
	}
}

// StartReplica initializes and starts the Raft replica on the NodeHost.
// It configures the replica based on RaftNode fields, sets up WAL and NodeHost directories,
// creates a new NodeHost instance if not already present (though typically NH is created by InitRaftNode),
// and then starts the on-disk replica.
//
// Returns:
//   - An error if any step fails, such as directory creation, NodeHost initialization, or starting the replica.
func (mn *RaftNode) StartReplica(NH *dragonboat.NodeHost) error {
	if runtime.GOOS == "darwin" {
		signal.Ignore(syscall.Signal(0xd))
	}
	raftParams := RecommendRaftParamsForShards()
	cfg := config.Config{
		ReplicaID:          mn.ReplicaID,
		ShardID:            mn.ShardID,
		CheckQuorum:        true,
		ElectionRTT:        uint64(raftParams.ElectionRTT),
		HeartbeatRTT:       uint64(raftParams.HeartbeatRTT),
		SnapshotEntries:    raftParams.SnapshotEntries,
		CompactionOverhead: raftParams.CompactionOverhead,
		IsNonVoting:        !ContainsRole(mn.Roles, RoleConsensus),
	}

	// stateMachine := func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	// 	return NewMasterKVRocksDBStateMachine(clusterID, nodeID)
	// }

	mn.NH = NH

	if !mn.Join && !IsMemberInMemberArray(mn.SelfMember, mn.InitialMembers) {
		return errors.New("the node itself must be inside initial-members")
	}

	initialMembersMap := ToInitialMembersMap(mn.InitialMembers)
	if mn.Join {
		initialMembersMap = map[uint64]string{}
	}
	log.Info().Interface("initialMembersMap", initialMembersMap).Interface("Join", mn.Join).Interface("cfg", cfg).Msg("Starting replica")
	return mn.NH.StartOnDiskReplica(initialMembersMap, mn.Join, mn.stateMachine, cfg)

}

// RequestAddReplica sends a request to the Raft cluster to add a new replica to the shard.
//
// Parameters:
//   - replicaID: The ID of the new replica to be added.
//   - member: The Member struct describing the new replica (e.g., its address).
//
// Returns:
//   - An error if the request to Dragonboat fails (e.g., timeout, node not leader).
//     The actual success/failure of adding the replica is logged based on the result channel.
func (mn *RaftNode) RequestAddReplica(replicaID uint64, member Member) error {
	addr := MemmberToAddr(member)
	// Request to add a new replica to the shard. The timeout is 3 seconds.
	// The ReplicaID of the new node is replicaID, its RaftAddress is addr.
	// The last argument 0 is the instance ID of the new replica, it is not used
	// by Dragonboat.
	rs, err := mn.NH.RequestAddReplica(mn.ShardID, replicaID, addr, 0, 3*time.Second)
	if err != nil {
		return err
	}
	// Wait for the result of the request.
	r := <-rs.ResultC()
	if r.Completed() {
		log.Info().Msg("✅ Replica added successfully") // Changed Spanish to English
	} else {
		log.Error().Interface("Result", r).Msg("❌ Error adding replica") // Changed Spanish to English
	}
	return err
}

// GetClient returns a client session (NoOPSession) for the shard this RaftNode belongs to.
// A NoOPSession is a light-weight session that can be used for proposing no-op entries
// or for certain types of read operations, but typically, for proposals that change state,
// a regular session from `ProposeSession` would be used.
//
// Returns:
//   - A *client.Session connected to the shard.
func (mn *RaftNode) GetClient() *client.Session {
	return mn.NH.GetNoOPSession(mn.ShardID)
}

func (mn *RaftNode) Stop() {
	mn.mu.Lock()
	defer mn.mu.Unlock()
	mn.stopped = true
}

// Write proposes a batch of commands to the Raft log.
// It marshals the commands into JSON and uses SyncPropose to apply them.
// This is a synchronous operation that waits for the proposal to be committed or to fail.
//
// Parameters:
//   - ctx: The context for the synchronous proposal.
//   - commands: A slice of Command objects to be written.
//
// Returns:
//   - The result of the proposal from the state machine.
//   - An error if marshaling fails or if SyncPropose encounters an error.
func (mn *RaftNode) Write(ctx context.Context, comand general_command.FSM_Command) (statemachine.Result, error) { // Changed FSM_Command to general_command.FSM_Command
	mn.mu.RLock() // Use read lock since we only read the stopped field
	defer mn.mu.RUnlock()
	if mn.stopped {
		return statemachine.Result{}, errors.New("raft node is stopped")
	}
	cs := mn.GetClient()

	var buf bytes.Buffer

	gob.NewEncoder(&buf).Encode(comand)
	return mn.NH.SyncPropose(ctx, cs, buf.Bytes())
}

// Read performs a linearizable local read from the Raft state machine using ReadIndex and ReadLocalNode.
func (mn *RaftNode) Read(ctx context.Context, cmd general_command.Query_Command) (interface{}, error) {
	if mn.stopped {
		return statemachine.Result{}, errors.New("raft node is stopped")
	}
	select {
	case mn.readSem <- struct{}{}:
		defer func() { <-mn.readSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(cmd); err != nil {
		return nil, err
	}

	// In Dragonboat v4, ReadLocalNode requires a RequestState from ReadIndex to ensure linearizability.
	rs, err := mn.NH.ReadIndex(mn.ShardID, 10*time.Second)
	if err != nil {
		return nil, err
	}

	select {
	case res := <-rs.ResultC():
		if !res.Completed() {
			return nil, errors.New("read index failed or timed out")
		}
		return mn.NH.ReadLocalNode(rs, buf.Bytes())
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (mn *RaftNode) SyncRead(ctx context.Context, cmd general_command.Query_Command) (interface{}, error) { // Changed Query_Command to general_command.Query_Command
	mn.mu.RLock() // Use read lock since we only read the stopped field
	defer mn.mu.RUnlock()
	if mn.stopped {
		return statemachine.Result{}, errors.New("raft node is stopped")
	}
	var buf bytes.Buffer

	gob.NewEncoder(&buf).Encode(cmd)

	result, err := mn.NH.SyncRead(ctx, mn.ShardID, buf.Bytes())
	return result, err
}

// StartNodeReadyWatcher starts a goroutine that periodically checks if the Raft node is ready
// to process proposals by attempting a synchronous proposal (SyncPropose) of a no-op like command.
// It sends true on the returned channel when the node becomes ready, and false if it becomes not ready.
// The goroutine can be stopped by canceling the provided context.
//
// Parameters:
//   - ctx: The context to control the lifetime of the watcher goroutine.
//   - interval: The time.Duration between readiness checks.
//
// Returns:
//   - A <-chan bool that emits true when the node is ready, and false otherwise.
//     The channel is closed when the watcher stops (either by context cancellation or error).
func (mn *RaftNode) StartNodeReadyWatcher(ctx context.Context, interval time.Duration) <-chan bool {
	readyChan := make(chan bool, 1) // Buffered to avoid blocking

	go func() {
		defer func() {
			// Protect against double-close with recover
			if r := recover(); r != nil {
				log.Debug().Interface("panic", r).Msg("Recovered from panic in StartNodeReadyWatcher")
			}
			// Safe close using a select with default
			select {
			case <-readyChan: // If already closed, this will not block
			default:
				close(readyChan)
			}
		}()

		var lastReady bool
		var initialized bool

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("NodeReadyWatcher stopped by context cancellation")
				return
			case <-ticker.C:
				// Check if node is stopped first
				mn.mu.RLock()
				if mn.stopped {
					mn.mu.RUnlock()
					log.Debug().Msg("NodeReadyWatcher stopped because node is stopped")
					return
				}
				mn.mu.RUnlock()

				checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)

				queryCmd := general_command.Query_Command{
					Now: time.Now().UnixNano(),
					Command: general_command.RK_Command{
						Key:                "readiness-check",
						Op:                 general_command.GetOp,
						ColumnFamilyName:   db.MetaFC,
						ColumnFamilySector: db.MetaFCSector,
					},
				}

				_, err := mn.Read(checkCtx, queryCmd)
				cancel()

				currentReady := (err == nil)
				if !initialized || currentReady != lastReady {
					lastReady = currentReady
					initialized = true
					select {
					case readyChan <- currentReady:
						// Message sent successfully
					case <-ctx.Done():
						// Context cancelled while trying to send
						return
					}
				}
			}
		}
	}()

	return readyChan
}

// InitRaftNode is a general function to initialize a new RaftNode.
// It sets up the RaftNode struct with the provided parameters and starts the replica.
// This function is often assigned to InitRaftNodeFunc to allow mocking during tests.
//
// Parameters:
//   - ShardID: The ID of the shard (Raft consensus group).
//   - ReplicaID: The unique ID for this replica within the shard.
//   - selfMember: A Member struct describing the current node.
//   - initialMembers: A slice of Member structs for bootstrapping a new shard.
//   - join: A boolean indicating if this node is joining an existing shard.
//   - roles: A slice of NodeRole defining the roles for this node.
//   - stateMachineFn: A factory function to create the on-disk state machine for this replica.
//     Note: The current implementation within InitRaftNode directly uses NewMasterKVRocksDBStateMachine,
//     effectively overriding the passed stateMachineFn if it were different. This might be an oversight
//     or intentional design for this specific InitRaftNode function.
//
// Returns:
//   - A pointer to the initialized and started RaftNode.
//   - An error if starting the replica fails.
func InitRaftNode(ShardID uint64, ReplicaID uint64, selfMember Member, initialMembers []Member, join bool, roles []NodeRole, NH *dragonboat.NodeHost, stateMachineFn func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine) (*RaftNode, error) {
	raftNode := &RaftNode{}
	raftNode.ReplicaID = ReplicaID
	raftNode.ShardID = ShardID
	raftNode.SelfMember = selfMember
	raftNode.InitialMembers = initialMembers
	raftNode.Join = join
	raftNode.Roles = roles
	raftNode.readSem = make(chan struct{}, 100) // max 100 inflight reads
	// The passed stateMachineFn is assigned here.
	raftNode.stateMachine = stateMachineFn
	// However, StartReplica currently uses its own hardcoded state machine if not careful.
	// The current RaftNode.StartReplica uses mn.stateMachine, so this assignment is correct.
	// The original comment in StartReplica about NewMasterKVRocksDBStateMachine might be misleading
	// if stateMachineFn is indeed passed and used.
	// The InitRaftNode was previously hardcoding `NewMasterKVRocksDBStateMachine`
	// It should use the provided `stateMachineFn`.
	// Corrected: It seems the InitRaftNode itself was previously hardcoding the state machine type
	// rather than using the passed `stateMachineFn`. The RaftNode struct correctly stores `stateMachineFn`
	// as `stateMachine` and `StartReplica` uses `mn.stateMachine`.
	// The specific call to `InitRaftNode` from `InitMasterNode` passes `NewMasterKVRocksDBStateMachine`.
	// The specific call from `InitTenantNode` (presumably in tenant-node.go) would pass `NewTenantKVRocksDBStateMachine`.
	// So, the `stateMachineFn` parameter IS being used correctly to instantiate the appropriate SM.

	err := raftNode.StartReplica(NH)
	if err != nil {
		return nil, err
	}

	return raftNode, nil
}
