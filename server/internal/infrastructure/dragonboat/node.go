package dragonboat

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/utils"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/client"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/statemachine"
	"github.com/rs/zerolog/log"
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
}

// StartReplica initializes and starts the Raft replica on the NodeHost.
// It configures the replica based on RaftNode fields, sets up WAL and NodeHost directories,
// creates a new NodeHost instance if not already present (though typically NH is created by InitRaftNode),
// and then starts the on-disk replica.
//
// Returns:
//   - An error if any step fails, such as directory creation, NodeHost initialization, or starting the replica.
func (mn *RaftNode) StartReplica() error {

	cfg := config.Config{
		ReplicaID:          mn.ReplicaID,
		ShardID:            mn.ShardID,
		CheckQuorum:        true,
		ElectionRTT:        10,
		HeartbeatRTT:       1,
		SnapshotEntries:    1000,
		CompactionOverhead: 500,
		IsNonVoting:        !ContainsRole(mn.Roles, RoleConsensus),
	}

	// stateMachine := func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	// 	return NewMasterKVRocksDBStateMachine(clusterID, nodeID)
	// }

	base_path, err := db.DefaultPathProvider{}.GetDatabasePath()
	if err != nil {
		return err
	}

	log.Info().Msg(base_path + "/wal/" + strconv.FormatUint(mn.ReplicaID, 10) + "/" + strconv.Itoa(mn.SelfMember.Port))
	mn.NH, err = dragonboat.NewNodeHost(config.NodeHostConfig{
		WALDir:         base_path + "/wal/" + strconv.FormatUint(mn.ReplicaID, 10) + "/" + mn.SelfMember.IP + "-" + strconv.Itoa(mn.SelfMember.Port),
		NodeHostDir:    base_path + "/node/" + strconv.FormatUint(mn.ReplicaID, 10) + "/" + mn.SelfMember.IP + "-" + strconv.Itoa(mn.SelfMember.Port),
		RTTMillisecond: 200,
		RaftAddress:    MemmberToAddr(mn.SelfMember),
	})
	if err != nil {
		return err
	}

	if !mn.Join && !IsMemberInMemberArray(mn.SelfMember, mn.InitialMembers) {
		return errors.New("the node itself must be inside initial-members")
	}

	initialMembersMap := ToInitialMembersMap(mn.InitialMembers)
	if mn.Join {
		initialMembersMap = map[uint64]string{}
	}
	fmt.Println("StartOnDiskReplica!!!", mn.stateMachine)
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
	select {
	case r := <-rs.ResultC():
		if r.Completed() {
			log.Info().Msg("✅ Replica added successfully") // Changed Spanish to English
		} else {
			log.Error().Interface("Result", r).Msg("❌ Error adding replica") // Changed Spanish to English
		}
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
func (mn *RaftNode) Write(ctx context.Context, comands []Command) (statemachine.Result, error) {
	cs := mn.GetClient()
	data, err := json.Marshal(comands)
	if err != nil {
		return statemachine.Result{}, err
	}
	result, err := mn.NH.SyncPropose(ctx, cs, data)
	return result, err
}

// Read performs a linearizable read from the Raft state machine.
// It marshals the read command (RK_Command) into JSON and uses SyncRead.
// This is a synchronous operation that waits for the read to be processed or to fail.
//
// Parameters:
//   - ctx: The context for the synchronous read.
//   - cmd: The RK_Command describing the read operation.
//
// Returns:
//   - The result of the read operation from the state machine.
//   - An error if marshaling fails or if SyncRead encounters an error.
func (mn *RaftNode) Read(ctx context.Context, cmd RK_Command) (interface{}, error) {
	data, err := json.Marshal(cmd)
	if err != nil {
		return statemachine.Result{}, err
	}
	result, err := mn.NH.SyncRead(ctx, mn.ShardID, data)
	return result, err
}

// StartNodeReadyWatcher starts a goroutine that periodically checks if the Raft node is ready
// to process proposals by attempting a synchronous proposal (SyncPropose) of a no-op like command.
// It sends true on the returned channel when the node becomes ready, and false if it becomes not ready.
// This can be used to monitor the health and leadership status of the node.
//
// Parameters:
//   - interval: The time.Duration between readiness checks.
//
// Returns:
//   - A <-chan bool that emits true when the node is ready, and false otherwise.
//     The channel is closed when the watcher stops (which is implicitly when the RaftNode might be stopped,
//     though this watcher runs indefinitely until the underlying NH operations fail consistently or the program exits).
func (mn *RaftNode) StartNodeReadyWatcher(interval time.Duration) <-chan bool {
	readyChan := make(chan bool)

	go func() {
		defer close(readyChan)
		var lastReady bool
		var initialized bool

		for {
			session := mn.NH.GetNoOPSession(mn.ShardID)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			var buf bytes.Buffer

			cmd := Command{
				Now:  utils.GetNowInInt(),
				Type: RW,
				CMD: RWK_Command{
					Op: Write,
					CMD: WK_Command{
						Key:              "ready",
						Value:            []byte(Int64ToBytes(time.Now().UnixMilli())),
						ColumnFamilyName: db.MetaFC,
						Op:               PutOp,
					},
				},
			}

			gob.NewEncoder(&buf).Encode(cmd)
			_, err := mn.NH.SyncPropose(ctx, session, buf.Bytes())

			cancel()

			currentReady := (err == nil)
			if !initialized || currentReady != lastReady {
				lastReady = currentReady
				initialized = true
				readyChan <- currentReady
			}

			time.Sleep(interval)
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
func InitRaftNode(ShardID uint64, ReplicaID uint64, selfMember Member, initialMembers []Member, join bool, roles []NodeRole, stateMachineFn func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine) (*RaftNode, error) {
	raftNode := &RaftNode{}
	raftNode.ReplicaID = ReplicaID
	raftNode.ShardID = ShardID
	raftNode.SelfMember = selfMember
	raftNode.InitialMembers = initialMembers
	raftNode.Join = join
	raftNode.Roles = roles
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

	err := raftNode.StartReplica()
	if err != nil {
		return nil, err
	}

	return raftNode, nil
}
