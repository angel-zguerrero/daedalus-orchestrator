package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	node_scheduler "deadalus-orch/server/internal/usecase/command/node-scheduler"
	"deadalus-orch/shared/models"
	"fmt"
	"time"
)

type NodeSchedulerBalancingBO struct {
	Config *common.ServerConfing
}

func NewNodeSchedulerBalancingBO(Config *common.ServerConfing) *NodeSchedulerBalancingBO {
	return &NodeSchedulerBalancingBO{
		Config: Config,
	}
}

func (bo *NodeSchedulerBalancingBO) GetState(ctx context.Context) (*models.NodeSchedulerBalancingState, error) {
	getStateCommand := &node_scheduler.GetNodeSchedulerBalancingStateCommand{}

	state, err := dragonboat.ExecuteRepositoryQuery[models.NodeSchedulerBalancingState](
		bo.Config.MasterNode,
		ctx,
		getStateCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"get node scheduler balancing state",
	)
	if err != nil {
		return nil, fmt.Errorf("get node scheduler balancing state command failed: %w", err)
	}

	return &state, nil
}

func (bo *NodeSchedulerBalancingBO) UpsertState(ctx context.Context, state models.NodeSchedulerBalancingState) error {
	upsertCommand := &node_scheduler.UpsertNodeSchedulerBalancingStateCommand{
		State: state,
	}

	_, err := dragonboat.ExecuteRepositoryCommand[bool](
		bo.Config.MasterNode,
		ctx,
		upsertCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"upsert node scheduler balancing state",
	)
	return err
}

func (bo *NodeSchedulerBalancingBO) BalanceNodeSchedulers() error {
	nodeSchedulerBO := NewNodeSchedulerBO(bo.Config)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	pageSize := 100
	cursor := ""
	connectedCount := 0
	totalCount := 0

	bo.Config.Logger.Info().Msg("⚖️ Starting Node Scheduler balancing process...")

	for {
		findResult, err := nodeSchedulerBO.GetNodeSchedulers(ctx, "", cursor, pageSize)
		if err != nil {
			return fmt.Errorf("failed to fetch node schedulers: %w", err)
		}

		totalCount += len(findResult.Entities)
		for _, ns := range findResult.Entities {
			if ns.ConnectionStatus == models.ConnectionStatusConnected {
				connectedCount++
			}
		}

		if findResult.Cursor == "" || len(findResult.Entities) < pageSize {
			break
		}
		cursor = findResult.Cursor
	}

	bo.Config.Logger.Info().
		Int("connected", connectedCount).
		Int("total", totalCount).
		Msg("⚖️ Node Schedulers balancing summary")

	fmt.Printf("Connected Node Schedulers: %d\n", connectedCount)

	return nil
}
