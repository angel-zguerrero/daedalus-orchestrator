package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	tentant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
	"errors"
	"fmt"
	"strings"

	"deadalus-orch/server/internal/infrastructure/dragonboat"
)

// DashboardSummaryBO provides business logic for retrieving the global dashboard summary.
type DashboardSummaryBO struct {
	Config *common.ServerConfing
}

func NewDashboardSummaryBO(config *common.ServerConfing) *DashboardSummaryBO {
	return &DashboardSummaryBO{Config: config}
}

// GetDashboardSummary retrieves the pre-aggregated global DashboardSummary from the master node.
func (bo *DashboardSummaryBO) GetDashboardSummary(ctx context.Context) (models.DashboardSummary, error) {
	getDashboardSummaryCommand := &tentant_command.GetDashboardSummaryCommand{}

	summary, err := dragonboat.ExecuteRepositoryQuery[models.DashboardSummary](
		bo.Config.MasterNode,
		ctx,
		getDashboardSummaryCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"get dashboard summary",
	)
	if err != nil {
		if strings.Contains(err.Error(), "entity not found") {
			return models.DashboardSummary{}, errors.New("DashboardSummary not found")
		}
		return models.DashboardSummary{}, fmt.Errorf("get dashboard summary command failed: %w", err)
	}

	return summary, nil
}
