package dragonboat

// InitRaftNodeFunc is a variable that can be replaced by a mock in tests.
var InitRaftNodeFunc = InitRaftNode

func InitMasterNode(ReplicaID uint64, selfMember Member, initialMembers []Member, join bool, roles []NodeRole) (*RaftNode, error) {
	return InitRaftNodeFunc(uint64(MasterShardID), ReplicaID, selfMember, initialMembers, join, roles, NewMasterKVRocksDBStateMachine)
}
