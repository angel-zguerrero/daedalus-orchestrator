package business_logic

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"fmt"

	"deadalus-orch/server/internal/pkg/config"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	tenant_summary_command "deadalus-orch/server/internal/usecase/command/tenant-summary"
	tenant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
	"encoding/gob"
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

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: findTenantCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.MasterNode.Read(readCtx, *queryCommand)
	if err != nil {
		if strings.Contains(err.Error(), "cannot encode nil pointer of type") {
			return models.TenantInMaster{}, nil, nil, errors.New("Tenant not found")
		}
		bo.Config.Logger.Error().Err(err).Msg("Find tenants command failed")
		return models.TenantInMaster{}, nil, nil, errors.New("Find tenants command failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Find tenants command failed")
		return models.TenantInMaster{}, nil, nil, errors.New("Find tenants command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find tenants command failed")
		return models.TenantInMaster{}, nil, nil, errors.New("Find tenants command failed")
	}

	if parsedResult.Result == nil {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find tenants command failed")
		return models.TenantInMaster{}, nil, nil, errors.New("Tenant not found")
	}

	tenantInMaster := parsedResult.Result.(models.TenantInMaster)
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

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: paginateTenantsCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.MasterNode.Read(readCtx, *queryCommand)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate tenants command failed")
		return db.FindResult[models.TenantInMaster]{}, errors.New("Login failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate tenants command failed")
		return db.FindResult[models.TenantInMaster]{}, errors.New("Paginate tenants command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Paginate tenants command failed")
		return db.FindResult[models.TenantInMaster]{}, errors.New("Paginate tenants command failed")
	}

	findResult := parsedResult.Result.(db.FindResult[models.TenantInMaster])
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

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: getTenantSummaryCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.MasterNode.Read(readCtx, *queryCommand)
	if err != nil {
		if strings.Contains(err.Error(), "cannot encode nil pointer of type") {
			return models.TenantSummary{}, errors.New("Tenant summary not found")
		}
		bo.Config.Logger.Error().Err(err).Msg("Get tenant summary command failed")
		return models.TenantSummary{}, errors.New("Get tenant summary command failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Get tenant summary command failed")
		return models.TenantSummary{}, errors.New("Get tenant summary command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Str("error", parsedResult.Error).Msg("Get tenant summary command failed")
		return models.TenantSummary{}, errors.New("Get tenant summary command failed: " + parsedResult.Error)
	}

	if parsedResult.Result == nil {
		bo.Config.Logger.Error().Str("error", parsedResult.Error).Msg("Get tenant summary command failed")
		return models.TenantSummary{}, errors.New("Tenant summary not found")
	}

	tenantSummary := parsedResult.Result.(models.TenantSummary)
	return tenantSummary, nil
}
