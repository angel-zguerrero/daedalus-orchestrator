package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"fmt"

	"deadalus-orch/server/internal/pkg/config"
	tenant_summary_command "deadalus-orch/server/internal/usecase/command/tenant-summary"
	tenant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
	"errors"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/dragonboat"

	"github.com/google/uuid"
	db4 "github.com/lni/dragonboat/v4"
)

type TenantBO struct {
	Config *common.ServerConfing
}

func NewTenantBO(Config *common.ServerConfing) *TenantBO {
	return &TenantBO{
		Config: Config,
	}
}

func (bo *TenantBO) SetTenantNode(shardID int, tenantId string) *dragonboat.RaftNode {
	var tenantNode *dragonboat.RaftNode

	bo.Config.TenantNodesLock.Lock()
	for i := range bo.Config.TenantNodes {
		if bo.Config.TenantNodes[i].ShardID == uint64(shardID) {
			tenantNode = bo.Config.TenantNodes[i]
			break
		}
	}
	bo.Config.TenantNodesLock.Unlock()

	bo.Config.TenantNodesLock.Lock()
	bo.Config.TenantNodesDictionary[tenantId] = tenantNode
	bo.Config.TenantNodesLock.Unlock()
	return tenantNode
}
func (bo *TenantBO) CreateTenant(ctx context.Context, code, name string) (models.TenantInMaster, error) {
	tenant := &models.TenantInMaster{
		ID:   strings.ReplaceAll(uuid.New().String(), "-", ""),
		Code: code,
		Name: name,
	}

	createdList, err := bo.BulkCreateTenant(ctx, []*models.TenantInMaster{tenant})
	if err != nil {
		return models.TenantInMaster{}, err
	}
	return createdList[0], nil
}

func (bo *TenantBO) BulkCreateTenant(ctx context.Context, tenants []*models.TenantInMaster) ([]models.TenantInMaster, error) {
	if len(tenants) == 0 {
		return nil, errors.New("no tenants provided")
	}

	// Asegurar IDs válidos
	for _, t := range tenants {
		if t.ID == "" {
			t.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
		}
	}

	createTenantInMasterCommand := &tenant_command.CreateTenantInMasterCommand{
		Tenants: make([]models.TenantInMaster, len(tenants)),
	}
	for i, t := range tenants {
		createTenantInMasterCommand.Tenants[i] = *t
	}

	// Using the new generic function instead of repetitive code
	timeout := config.GlobalConfiguration.ApiRaftTimeout * time.Duration(len(tenants))
	created, err := dragonboat.ExecuteRepositoryCommand[[]models.TenantInMaster](
		bo.Config.MasterNode,
		ctx,
		createTenantInMasterCommand,
		timeout,
		bo.Config.Logger,
		"bulk tenant creation",
	)
	if err != nil {
		return nil, err
	}

	// Crear ColumnFamilies y recolectar códigos
	var tenantCodes []string
	for i := range created {
		tenantNode := bo.SetTenantNode(created[i].ShardId, created[i].ID)
		if tenantNode == nil {
			return nil, fmt.Errorf("tenant node not found for ID %s", created[i].ID)
		}

		tenantCodes = append(tenantCodes, created[i].Code)
	}

	// Asignar todos los tenants a shard en un solo comando
	assignCmd := &tenant_command.AssignToShardTenantInMasterCommand{TenantCodes: tenantCodes}

	// Using the new generic function for the assignment command
	assigned, err := dragonboat.ExecuteRepositoryCommand[bool](
		bo.Config.MasterNode,
		ctx,
		assignCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"shard assignment for tenants",
	)
	if err != nil {
		return nil, err
	}

	if assigned {
		for i := range created {
			created[i].Status = models.Assigned
			bo.Config.Logger.Info().Str("code", created[i].Code).Msg("tenant asserted successfully")
		}
	}

	return created, nil
}

func (bo *TenantBO) GetTenant(ctx context.Context, tenantCode string) (models.TenantInMaster, *dragonboat.RaftNode, *db4.NodeHostInfo, error) {
	findTenantCommand := &tenant_command.FindTenantCommand{
		TenantCode: tenantCode,
	}

	tenantInMaster, err := dragonboat.ExecuteRepositoryQuery[models.TenantInMaster](
		bo.Config.MasterNode,
		ctx,
		findTenantCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"find tenant",
	)
	if err != nil {
		if strings.Contains(err.Error(), "entity not found") {
			return models.TenantInMaster{}, nil, nil, errors.New("Tenant not found")
		}
		return models.TenantInMaster{}, nil, nil, fmt.Errorf("find tenant failed: %w", err)
	}

	node := bo.Config.TenantNodesDictionary[tenantInMaster.ID]

	if node == nil {
		return tenantInMaster, nil, nil, nil
	}

	nodeHostInfoOption := &db4.NodeHostInfoOption{SkipLogInfo: true}
	nodeHostInfo := node.NH.GetNodeHostInfo(*nodeHostInfoOption)
	return tenantInMaster, node, nodeHostInfo, nil
}

func (bo *TenantBO) DeleteTenant(ctx context.Context, tenantCode string) error {
	markToDeletionTenantInMasterCommand := &tenant_command.MarkToDeletionTenantInMasterCommand{
		TenantCode: tenantCode,
	}

	// Using the new generic function instead of repetitive code
	_, err := dragonboat.ExecuteRepositoryCommand[interface{}](
		bo.Config.MasterNode,
		ctx,
		markToDeletionTenantInMasterCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"tenant deletion",
	)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	return nil
}

func (bo *TenantBO) GetTenants(ctx context.Context, q string, cursor string, pageSize int) (db.FindResult[models.TenantInMaster], error) {
	paginateTenantsCommand := &tenant_command.PaginateTenantsCommand{
		Cursor:   cursor,
		PageSize: pageSize,
		Q:        q,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.TenantInMaster]](
		bo.Config.MasterNode,
		ctx,
		paginateTenantsCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"paginate tenants",
	)
	if err != nil {
		return db.FindResult[models.TenantInMaster]{}, fmt.Errorf("paginate tenants failed: %w", err)
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.TenantInMaster{}
	}

	return findResult, nil
}

func (bo *TenantBO) GetTenantSummary(ctx context.Context, tenantCode string) (models.TenantSummary, error) {
	tenant, _, _, err := bo.GetTenant(ctx, tenantCode)
	if err != nil {
		bo.Config.Logger.Error().Str("error", err.Error()).Msg("Get tenant summary command failed")
		return models.TenantSummary{}, err
	}

	getTenantSummaryCommand := &tenant_summary_command.GetTenantSummaryCommand{
		TenantId: tenant.ID,
	}

	tenantSummary, err := dragonboat.ExecuteRepositoryQuery[models.TenantSummary](
		bo.Config.MasterNode,
		ctx,
		getTenantSummaryCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"get tenant summary",
	)
	if err != nil {
		if strings.Contains(err.Error(), "entity not found") {
			return models.TenantSummary{}, errors.New("Tenant summary not found")
		}
		return models.TenantSummary{}, fmt.Errorf("get tenant summary failed: %w", err)
	}

	return tenantSummary, nil
}
