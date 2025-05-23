package dragonboat

func InitMasterNode(ReplicaID uint64, selfMember Member, initialMembers []Member, join bool, roles []NodeRole) (*RaftNode, error) {
	return InitRaftNode(uint64(MasterShardID), ReplicaID, selfMember, initialMembers, join, roles, NewMasterKVRocksDBStateMachine)
}
