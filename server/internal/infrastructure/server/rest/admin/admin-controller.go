package rest_api_admin

import (
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
)

// AdminController handles the administrative REST API endpoints.
type AdminController struct {
	Config *common.RestServerConfing
}

// NewAdminController creates a new instance of RestAdminAPI.
func NewAdminController(Config *common.RestServerConfing) *AdminController {

	api := &AdminController{
		Config: Config,
	}

	return api
}

func (c *AdminController) SetTenantNode(shardID int, tenantId string) *dragonboat.RaftNode {
	var tenant *dragonboat.RaftNode

	c.Config.TenantNodesLock.Lock()
	for i := range c.Config.TenantNodes {
		if c.Config.TenantNodes[i].ShardID == uint64(shardID) {
			tenant = c.Config.TenantNodes[i]
			break
		}
	}
	c.Config.TenantNodesLock.Unlock()

	c.Config.TenantNodesLock.Lock()
	c.Config.TenantNodesDictionary[tenantId] = tenant
	c.Config.TenantNodesLock.Unlock()
	return tenant
}
