package tenant

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/server/common"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/tenant"
	"deadalus-orch/server/internal/pkg/config"

	commands "deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
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
