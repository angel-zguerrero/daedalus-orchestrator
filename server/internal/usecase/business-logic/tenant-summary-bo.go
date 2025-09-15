package business_logic

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/server/common"

	"deadalus-orch/server/internal/pkg/config"
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

func (bo *TenantSummaryBO) GetTenantSummary(ctx context.Context, tenantId, cf, cfs string) (models.TenantSummary, error) {
	getTenantSummaryCommand := &tenant_summary_command.GetTenantSummaryCommand{
		TenantId: tenantId,
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
		bo.Config.Logger.Error().Str("tenantId", tenantId).Msg("Tenant summary not found")
		return models.TenantSummary{}, errors.New("TenantSummary not found")
	}

	tenantSummary := parsedResult.Result.(models.TenantSummary)

	return tenantSummary, nil
}
