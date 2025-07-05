package dragonboat

import "deadalus-orch/server/internal/infrastructure/db"

var InitTenantRaftNodeFunc = InitRaftNode

func InitTenantNode(ShardID, ReplicaID uint64, selfMember Member, initialMembers []Member, join bool, roles []NodeRole, pathProvider db.PathProvider) (*RaftNode, error) {
	return InitTenantRaftNodeFunc(ShardID, ReplicaID, selfMember, initialMembers, join, roles, NewTenantKVStateMachine(pathProvider))
}
