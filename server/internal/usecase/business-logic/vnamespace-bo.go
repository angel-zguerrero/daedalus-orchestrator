package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	vnamespace_command "deadalus-orch/server/internal/usecase/command/vnamespace"
	"deadalus-orch/shared/models"
	"fmt"
)

type VNamespaceBO struct {
	Config *common.ServerConfing
}

func NewVNamespaceBO(config *common.ServerConfing) *VNamespaceBO {
	return &VNamespaceBO{
		Config: config,
	}
}

func (bo *VNamespaceBO) GetVNamespaces(ctx context.Context, q string, cursor string, pageSize int, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) (db.FindResult[models.VNamespace], error) {
	paginateVNamespacesCommand := &vnamespace_command.PaginateVNamespacesCommand{
		Query:    q,
		Cursor:   cursor,
		PageSize: pageSize,
		CF:       cf,
		CFS:      cfs,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.VNamespace]](
		tenantNode,
		ctx,
		paginateVNamespacesCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"paginate vnamespaces",
	)
	if err != nil {
		return db.FindResult[models.VNamespace]{}, fmt.Errorf("paginate vnamespaces failed: %w", err)
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.VNamespace{}
	}

	return findResult, nil
}

// GetVNamespacesWithFilter paginates vnamespaces using DB-level filter rules derived from a
// ClaimWorkFilter. Inclusion lists, exact exclusions, and LIKE patterns are pushed to the
// repository; ExcludeVNamespacePatterns are applied in memory inside the repository.
func (bo *VNamespaceBO) GetVNamespacesWithFilter(ctx context.Context, filter models.ClaimWorkFilter, cursor string, pageSize int, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) (db.FindResult[models.VNamespace], error) {
	cmd := &vnamespace_command.PaginateVNamespacesWithFilterCommand{
		Filter:   filter,
		Cursor:   cursor,
		PageSize: pageSize,
		CF:       cf,
		CFS:      cfs,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.VNamespace]](
		tenantNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"paginate vnamespaces with filter",
	)
	if err != nil {
		return db.FindResult[models.VNamespace]{}, fmt.Errorf("paginate vnamespaces with filter failed: %w", err)
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.VNamespace{}
	}

	return findResult, nil
}
