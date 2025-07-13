package business_logic

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"

	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
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
	var tenant *dragonboat.RaftNode

	bo.Config.TenantNodesLock.Lock()
	for i := range bo.Config.TenantNodes {
		if bo.Config.TenantNodes[i].ShardID == uint64(shardID) {
			tenant = bo.Config.TenantNodes[i]
			break
		}
	}
	bo.Config.TenantNodesLock.Unlock()

	bo.Config.TenantNodesLock.Lock()
	bo.Config.TenantNodesDictionary[tenantId] = tenant
	bo.Config.TenantNodesLock.Unlock()
	return tenant
}

func (bo *TenantBO) CreateTenant(ctx context.Context, code, name string) (models.TenantInMaster, error) {
	createTenantInMasterCommand := &tenant_command.CreateTenantInMasterCommand{
		TenantId:   strings.ReplaceAll(uuid.New().String(), "-", ""),
		TenantCode: code,
		TenantName: name,
	}

	fsmCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  createTenantInMasterCommand,
	}

	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer writeCancel()

	result, err := bo.Config.MasterNode.Write(writeCtx, fsmCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Str("Code", code).Msg("Failed to create new tenant")
		return models.TenantInMaster{}, errors.New("Failed to create new tenant: " + err.Error())
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Str("Code", code).Msg("Tenant creation command returned unexpected result type")
		return models.TenantInMaster{}, errors.New("Tenant creation command returned unexpected error")
	}

	if parsedResult.Error != "" {
		return models.TenantInMaster{}, errors.New("Failed to create new tenant error: " + parsedResult.Error)
	}

	tenantInMaster := parsedResult.Result.(models.TenantInMaster)
	tenantNode := bo.SetTenantNode(tenantInMaster.ShardId, tenantInMaster.ID)

	if tenantNode == nil {
		return models.TenantInMaster{}, errors.New("Tenant node not found")
	}

	createColumnFamilyCommand := &commands.CreateColumnFamilyCommand{
		Name: tenantInMaster.ID,
	}

	ccfCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  createColumnFamilyCommand,
	}

	result, err = tenantNode.Write(writeCtx, ccfCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Str("Code", code).Msg("Failed to create new tenant")
		return models.TenantInMaster{}, errors.New("Failed to create new tenant: " + err.Error())
	}

	buf = bytes.NewBuffer(result.Data)
	dec = gob.NewDecoder(buf)

	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Str("Code", code).Msg("Tenant creation command returned unexpected result type")
		return models.TenantInMaster{}, errors.New("Tenant creation command returned unexpected error")
	}

	if parsedResult.Error != "" {
		return models.TenantInMaster{}, errors.New("Failed to create new tenant error: " + parsedResult.Error)
	}

	assignToShardTenantInMasterCommand := &tenant_command.AssignToShardTenantInMasterCommand{
		TenantCode: code,
	}

	atstCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  assignToShardTenantInMasterCommand,
	}

	result, err = bo.Config.MasterNode.Write(writeCtx, atstCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Str("Code", code).Msg("Failed to create new tenant")
		return models.TenantInMaster{}, errors.New("Failed to create new tenant: " + err.Error())
	}

	buf = bytes.NewBuffer(result.Data)
	dec = gob.NewDecoder(buf)

	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Str("Code", code).Msg("Tenant creation command returned unexpected result type")
		return models.TenantInMaster{}, errors.New("Tenant creation command returned unexpected error")
	}

	if parsedResult.Error != "" {
		return models.TenantInMaster{}, errors.New("Failed to create new tenant error: " + parsedResult.Error)
	}

	if parsedResult.Result.(bool) {
		tenantInMaster.Status = models.Assigned
	}

	bo.Config.Logger.Info().Str("code", code).Msg("tenant asserted successfully")
	return tenantInMaster, nil
}

func (bo *TenantBO) GetTenant(ctx context.Context, tenantID string) (models.TenantInMaster, *dragonboat.RaftNode, *db4.NodeHostInfo, error) {
	findTenantCommand := &tenant_command.FindTenantCommand{
		TenantID: tenantID,
	}

	queryCommand := &commands.Query_Command{
		Command: &commands.Repository_Command{
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

func (bo *TenantBO) DeleteTenant(ctx context.Context, tenantID string) error {
	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer writeCancel()

	markToDeletionTenantInMasterCommand := &tenant_command.MarkToDeletionTenantInMasterCommand{
		TenantId: tenantID,
	}

	atstCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  markToDeletionTenantInMasterCommand,
	}

	result, err := bo.Config.MasterNode.Write(writeCtx, atstCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Str("TenantID", tenantID).Msg("Failed to delete tenant")
		return errors.New("Failed to delete tenant: " + err.Error())
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Str("TenantID", tenantID).Msg("Tenant deletion command returned unexpected result type")
		return errors.New("Tenant deletion command returned unexpected error")
	}

	if parsedResult.Error != "" {
		return errors.New("Failed to delete tenant error: " + parsedResult.Error)
	}

	deleteColumnFamilyCommand := &commands.DeleteColumnFamilyCommand{
		Name: tenantID,
	}

	ccfCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  deleteColumnFamilyCommand,
	}

	tenantNode := bo.Config.TenantNodesDictionary[tenantID]
	if tenantNode == nil {
		return errors.New("Failed to delete tenant error: Tenant node not found")
	}

	_, err = tenantNode.Write(writeCtx, ccfCmd)
	if err != nil {
		return errors.New("Failed to delete tenant error: " + err.Error())
	}

	deleteTenantInMasterCommand := &tenant_command.DeleteTenantInMasterCommand{
		TenantId: tenantID,
	}

	atstCmd = commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  deleteTenantInMasterCommand,
	}

	_, err = bo.Config.MasterNode.Write(writeCtx, atstCmd)
	if err != nil {
		return errors.New("Failed to delete tenant error: " + err.Error())
	}

	bo.Config.Logger.Info().Str("TenantID", tenantID).Msg("new tenant deleted successfully")
	return nil
}

func (bo *TenantBO) GetTenants(ctx context.Context, cursor string, pageSize int) (db.FindResult[models.TenantInMaster], error) {
	paginateTenantsCommand := &tenant_command.PaginateTenantsCommand{
		Cursor:   cursor,
		PageSize: pageSize,
	}

	queryCommand := &commands.Query_Command{
		Command: &commands.Repository_Command{
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
