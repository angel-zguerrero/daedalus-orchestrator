package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"

	"deadalus-orch/server/internal/pkg/config"
	tenant_summary_command "deadalus-orch/server/internal/usecase/command/tenant-summary"
	"deadalus-orch/shared/models"
	"errors"
	"fmt"
	"strings"
)

type TenantSummaryBO struct {
	Config *common.ServerConfing
}

func NewTenantSummaryBO(Config *common.ServerConfing) *TenantSummaryBO {
	return &TenantSummaryBO{
		Config: Config,
	}
}

func (bo *TenantSummaryBO) GetTenantSummary(ctx context.Context, tenantId, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) (models.TenantSummary, error) {
	getTenantSummaryCommand := &tenant_summary_command.GetTenantSummaryCommand{
		TenantId: tenantId,
	}

	tenantSummary, err := dragonboat.ExecuteRepositoryQuery[models.TenantSummary](
		tenantNode,
		ctx,
		getTenantSummaryCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"get tenant summary",
	)
	if err != nil {
		if strings.Contains(err.Error(), "entity not found") {
			return models.TenantSummary{}, errors.New("TenantSummary not found")
		}
		return models.TenantSummary{}, fmt.Errorf("get tenant summary command failed: %w", err)
	}

	return tenantSummary, nil
}
