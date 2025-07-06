package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"

	"github.com/lni/dragonboat/v4"
)

func StartTentantNodes(
	replicaID uint64,
	selfMember Member,
	join bool,
	roles []NodeRole,
	pathProvider db.PathProvider,
	initialMembers []Member,
	NH *dragonboat.NodeHost,
) ([]*RaftNode, error) {
	MaxTenants := config.GlobalConfiguration.MaxTenants

	var tenantNodes []*RaftNode

	for shardID := 0; shardID < MaxTenants; shardID++ {

		node, err := InitTenantNode(uint64(shardID+MasterShardID)+1, replicaID, selfMember, initialMembers, join, roles, NH, pathProvider)
		if err != nil {
			return nil, err
		}

		tenantNodes = append(tenantNodes, node)
	}

	return tenantNodes, nil
}
