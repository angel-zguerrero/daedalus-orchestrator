package tenant

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/tenant"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"

	commands "deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"

	"github.com/google/uuid"
)

type TenantService struct {
	pb.UnimplementedTenantServiceServer           // Embeds the unimplemented server for forward compatibility.
	startTime                           time.Time // The time when the TenantServer was instantiated, used to calculate uptime.
	Config                              *common.RestServerConfing
}

func NewTenantService(config *common.RestServerConfing) *TenantService {
	return &TenantService{
		startTime: time.Now(),
		Config:    config,
	}
}

func (c *TenantService) SetTenantNode(shardID int, tenantId string) *dragonboat.RaftNode {
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

func (ctrl *TenantService) AssertTenant(ctx context.Context, r *pb.AssertTenantRequest) (*pb.AssertTenantResponse, error) {
	createTenantInMasterCommand := &commands.CreateTenantInMasterCommand{
		TenantId:   strings.ReplaceAll(uuid.New().String(), "-", ""),
		TenantCode: r.Code,
		TenantName: r.Name,
	}

	fsmCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  createTenantInMasterCommand,
	}

	writeCtx, writeCancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout) // Or a specific timeout for writes
	defer writeCancel()

	result, err := ctrl.Config.MasterNode.Write(writeCtx, fsmCmd)
	if err != nil {

		ctrl.Config.Logger.Error().Err(err).Str("Code", r.Code).Msg("Failed to create new tenant")
		return nil, errors.New("Failed to create new tenant: " + err.Error())
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("Code", r.Code).Msg("Tenant creation command returned unexpected result type")
		return nil, errors.New("Tenant creation command returned unexpected error")
	}

	possibleErr := parsedResult.Error
	if possibleErr != "" {
		return nil, errors.New("Failed to create new tenant error: " + possibleErr)
	}

	tenantInMaster := parsedResult.Result.(models.TenantInMaster)

	tenantNode := ctrl.SetTenantNode(tenantInMaster.ShardId, tenantInMaster.ID)

	if tenantNode == nil {
		return nil, errors.New("Tenant node not found")
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

		ctrl.Config.Logger.Error().Err(err).Str("Code", r.Code).Msg("Failed to create new tenant")
		return nil, errors.New("Failed to create new tenant: " + err.Error())
	}

	buf = bytes.NewBuffer(result.Data)
	dec = gob.NewDecoder(buf)

	if err := dec.Decode(parsedResult); err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("Code", r.Code).Msg("Tenant creation command returned unexpected result type")
		return nil, errors.New("Tenant creation command returned unexpected error")
	}

	possibleErr = parsedResult.Error
	if possibleErr != "" {
		return nil, errors.New("Failed to create new tenant error: " + possibleErr)
	}

	assignToShardTenantInMasterCommand := &commands.AssignToShardTenantInMasterCommand{
		TenantCode: r.Code,
	}

	atstCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  assignToShardTenantInMasterCommand,
	}

	result, err = ctrl.Config.MasterNode.Write(writeCtx, atstCmd)
	if err != nil {

		ctrl.Config.Logger.Error().Err(err).Str("Code", r.Code).Msg("Failed to create new tenant")
		return nil, errors.New("Failed to create new tenant: " + err.Error())
	}

	buf = bytes.NewBuffer(result.Data)
	dec = gob.NewDecoder(buf)

	if err := dec.Decode(parsedResult); err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("Code", r.Code).Msg("Tenant creation command returned unexpected result type")
		return nil, errors.New("Tenant creation command returned unexpected error")
	}

	possibleErr = parsedResult.Error
	if possibleErr != "" {
		return nil, errors.New("Failed to create new tenant error: " + possibleErr)
	}

	if parsedResult.Result.(bool) {
		tenantInMaster.Status = models.Assigned
	}

	ctrl.Config.Logger.Info().Str("code", r.Code).Msg("tenant asserted successfully")
	return &pb.AssertTenantResponse{
		ID:        tenantInMaster.ID,
		Name:      tenantInMaster.Name,
		ShardId:   int64(tenantInMaster.ShardId),
		Code:      tenantInMaster.Code,
		Status:    string(tenantInMaster.Status),
		CreatedAt: tenantInMaster.CreatedAt.Format(time.RFC3339),
		UpdatedAt: tenantInMaster.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (ctrl *TenantService) GetTenantInfo(ctx context.Context, r *pb.TenantInfoRequest) (*pb.TenantInfoResponse, error) {
	tenantID := r.ID
	findTenantCommand := &commands.FindTenantCommand{
		TenantID: tenantID,
	}

	queryCommand := &commands.Query_Command{
		Command: &commands.Repository_Command{
			CMD: findTenantCommand,
		},
		Now: time.Now().UnixNano(), // Or handle as per specific requirements if Query_Command.Now is actively used
	}

	ctx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := ctrl.Config.MasterNode.Read(ctx, *queryCommand)
	if err != nil {
		if strings.Contains(err.Error(), "cannot encode nil pointer of type") {
			return nil, errors.New("tenant not found")
		}

		ctrl.Config.Logger.Error().Err(err).Msg("Find tenants command failed")

		return nil, errors.New("Find tenants command failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		ctrl.Config.Logger.Error().Err(err).Msg("Find tenants command failed")
		return nil, errors.New("Find tenants command failed: " + err.Error())
	}

	if parsedResult.Error != "" {
		ctrl.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find tenants command failed")
		return nil, errors.New("Find tenants command failed: " + parsedResult.Error)
	}

	if parsedResult.Result == nil {
		ctrl.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find tenants command failed")
		return nil, errors.New("tenant not found ")
	}

	tenantInMaster := parsedResult.Result.(models.TenantInMaster)

	return &pb.TenantInfoResponse{
		ID:        tenantInMaster.ID,
		ShardId:   int64(tenantInMaster.ShardId),
		Code:      tenantInMaster.Code,
		Status:    string(tenantInMaster.Status),
		CreatedAt: tenantInMaster.CreatedAt.Format(time.RFC3339),
		UpdatedAt: tenantInMaster.UpdatedAt.Format(time.RFC3339),
	}, nil
}
