package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
)

func StartTentantNodes(
	replicaID uint64,
	selfMember Member,
	initialMembers []Member,
	join bool,
	roles []NodeRole,
	pathProvider db.PathProvider,
) error {
	MaxTenants := config.GlobalConfiguration.MaxTenants
	for shardID := 0; shardID < MaxTenants; shardID++ {
		tenantMember := Member{
			IP: selfMember.IP,
			//Port: config.GlobalConfiguration.TenantPortLowerBound + shardID,
		}
		_, err := InitTenantNode(uint64(shardID+MasterShardID), replicaID, tenantMember, initialMembers, join, roles, pathProvider)
		if err != nil {
			return err
		}
	}
	return nil
}
