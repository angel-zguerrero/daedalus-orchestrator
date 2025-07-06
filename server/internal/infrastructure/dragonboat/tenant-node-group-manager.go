package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/shared/constants"
)

func StartTentantNodes(
	replicaID uint64,
	selfMember Member,
	join bool,
	roles []NodeRole,
	pathProvider db.PathProvider,
) ([]*RaftNode, error) {
	MaxTenants := config.GlobalConfiguration.MaxTenants

	var MaxReplicaId int
	if config.GlobalConfiguration.Env == string(constants.PRODUCTION) {
		MaxReplicaId = constants.MaxReplicationInProduction
	} else {
		MaxReplicaId = constants.MaxReplicationInNonProduction
	}

	localOffset := 0
	if config.GlobalConfiguration.Env != string(constants.PRODUCTION) {
		localOffset = constants.MaxReplicationInNonProduction
	}

	var tenantNodes []*RaftNode

	for shardID := 0; shardID < MaxTenants; shardID++ {
		tenantBasePort := shardID*localOffset + MaxReplicaId + 1 + config.GlobalConfiguration.ClusterBasePort + shardID
		port := tenantBasePort + int(replicaID)

		tenantMember := Member{
			IP:   selfMember.IP,
			Port: port,
		}

		initialMembers, err := ParseMembersFlag(&config.GlobalConfiguration.InitialMembers, tenantBasePort)
		if err != nil {
			return nil, err
		}

		node, err := InitTenantNode(uint64(shardID+MasterShardID)+1, replicaID, tenantMember, initialMembers, join, roles, pathProvider)
		if err != nil {
			return nil, err
		}

		tenantNodes = append(tenantNodes, node)
	}

	return tenantNodes, nil
}
