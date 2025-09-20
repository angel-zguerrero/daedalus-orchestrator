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

func (bo *VNamespaceBO) GetVNamespaces(ctx context.Context, q string, cursor string, pageSize int, cf, cfs string) (db.FindResult[models.VNamespace], error) {
	paginateVNamespacesCommand := &vnamespace_command.PaginateVNamespacesCommand{
		Query:    q,
		Cursor:   cursor,
		PageSize: pageSize,
		CF:       cf,
		CFS:      cfs,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.VNamespace]](
		bo.Config.TenantNodesDictionary[cfs],
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
