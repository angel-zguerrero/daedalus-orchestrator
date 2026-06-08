package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"

	"github.com/lni/dragonboat/v4"
)

// InitRaftNodeFunc is a function variable that points to InitRaftNode by default.
// It allows for replacing the actual Raft node initialization logic with a mock
// implementation during testing. This is a common pattern for dependency injection in Go tests.
var InitRaftNodeFunc = InitRaftNode

// InitMasterNode initializes a RaftNode specifically configured to act as the master node/shard in the cluster.
// The master shard has a predefined ID (MasterShardID).
// It utilizes the generic InitRaftNodeFunc (which defaults to InitRaftNode) to perform the actual
// Raft node setup, passing NewMasterKVStateMachine as the factory for the state machine.
//
// Parameters:
//   - ReplicaID: The unique ID for this replica within the Raft group for the master shard.
//   - selfMember: A Member struct describing the network address and other properties of the current node.
//   - initialMembers: A slice of Member structs representing the initial members of the Raft group for the master shard.
//     This is typically used when bootstrapping a new cluster.
//   - join: A boolean flag indicating whether this node should attempt to join an existing Raft group (true)
//     or participate in creating a new one (false).
//   - roles: A slice of NodeRole defining the roles this node will fulfill (e.g., consensus, scheduler).
//   - pathProvider: The PathProvider for determining database storage paths.
//   - sharedDBProvider: The SharedDBProvider instance shared across all shards.
//
// Returns:
//   - A pointer to the initialized RaftNode for the master shard.
//   - An error if the Raft node initialization fails.
func InitMasterNode(ReplicaID uint64, selfMember Member, initialMembers []Member, join bool, roles []NodeRole, pathProvider db.PathProvider, sharedDBProvider *db.SharedDBProvider, NH *dragonboat.NodeHost) (*RaftNode, error) {
	return InitRaftNodeFunc(uint64(MasterShardID), ReplicaID, selfMember, initialMembers, join, roles, NH, NewMasterKVStateMachine(pathProvider, sharedDBProvider))
}
