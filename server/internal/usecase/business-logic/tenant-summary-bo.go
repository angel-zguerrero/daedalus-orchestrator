package business_logic

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"fmt"

	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	tenant_summary_command "deadalus-orch/server/internal/usecase/command/tenant-summary"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"errors"
	"strings"
	"time"
)

type TenantSummaryBO struct {
	Config *common.ServerConfing
}

func NewTenantSummaryBO(Config *common.ServerConfing) *TenantSummaryBO {
	return &TenantSummaryBO{
		Config: Config,
	}
}

func (bo *TenantSummaryBO) GetTenantSummary(ctx context.Context, tenantID, cf, cfs string) (models.TenantSummary, error) {
	getTenantSummaryCommand := &tenant_summary_command.GetTenantSummaryCommand{
		TenantId: tenantID,
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: getTenantSummaryCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.TenantNodesDictionary[cfs].Read(readCtx, *queryCommand)
	if err != nil {
		if strings.Contains(err.Error(), "cannot encode nil pointer of type") {
			return models.TenantSummary{}, errors.New("TenantSummary not found")
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
		bo.Config.Logger.Error().Str("tenantID", tenantID).Msg("Tenant summary not found")
		return models.TenantSummary{}, errors.New("TenantSummary not found")
	}

	tenantSummary := parsedResult.Result.(models.TenantSummary)

	return tenantSummary, nil
}

func (bo *TenantSummaryBO) UpdateTenantSummary(ctx context.Context, tenantSummary *models.TenantSummary, cf, cfs string) (models.TenantSummary, error) {
	updateTenantSummaryCommand := &tenant_summary_command.UpdateTenantSummaryCommand{
		TenantSummary: *tenantSummary,
	}

	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer writeCancel()

	fsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  updateTenantSummaryCommand,
	}

	result, err := bo.Config.TenantNodesDictionary[cfs].Write(writeCtx, fsmCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Failed to update tenant summary")
		return models.TenantSummary{}, fmt.Errorf("failed to update tenant summary: %w", err)
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Update tenant summary command returned unexpected result type")
		return models.TenantSummary{}, fmt.Errorf("update tenant summary command returned decode error: %w", err)
	}

	if parsedResult.Error != "" {
		return models.TenantSummary{}, fmt.Errorf("update tenant summary failed: %s", parsedResult.Error)
	}

	updated := parsedResult.Result.(models.TenantSummary)

	return updated, nil
}
