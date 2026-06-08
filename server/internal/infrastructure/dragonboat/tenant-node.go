package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"

	"github.com/lni/dragonboat/v4"
)

var InitTenantRaftNodeFunc = InitRaftNode

func InitTenantNode(ShardID, ReplicaID uint64, selfMember Member, initialMembers []Member, join bool, roles []NodeRole, NH *dragonboat.NodeHost, pathProvider db.PathProvider, sharedDBProvider *db.SharedDBProvider) (*RaftNode, error) {
	return InitTenantRaftNodeFunc(ShardID, ReplicaID, selfMember, initialMembers, join, roles, NH, NewTenantKVStateMachine(pathProvider, sharedDBProvider))
}
