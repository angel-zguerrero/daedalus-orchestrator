package node_scheduler

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(UpsertNodeSchedulerCommand{})
	gob.Register(models.NodeScheduler{})
	gob.Register([]models.NodeScheduler{})
}

type UpsertNodeSchedulerCommand struct {
	NodeSchedulers []models.NodeScheduler
}

func (cmd *UpsertNodeSchedulerCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	exchangeRepo, err := db.NewNodeSchedulerRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var resultNodeSchedulers []models.NodeScheduler

	for _, nodeScheduler := range cmd.NodeSchedulers {

		// Validate that code is not empty
		if nodeScheduler.Name == "" {
			commandResult.Error = "NodeScheduler name is required"
			return *commandResult
		}

		// Look for existing exchange by code (primary upsert strategy)
		existing, err := exchangeRepo.GetNodeSchedulerByName(nodeScheduler.Name, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		nodeScheduler.TTL = config.GlobalConfiguration.NodeSchedulerTTL * 60 // Convert minutes to seconds

		var rebalanceNeeded bool
		if existing != nil {
			nodeScheduler.ID = existing.ID
			nodeScheduler.Name = existing.Name
			nodeScheduler.CreatedAt = existing.CreatedAt
			nodeScheduler.RunningStatus = existing.RunningStatus
			if nodeScheduler.BalancingId == "" {
				nodeScheduler.BalancingId = existing.BalancingId
			}

			if nodeScheduler.LastHeartbeat.IsZero() {
				nodeScheduler.LastHeartbeat = existing.LastHeartbeat
			}

			if nodeScheduler.LastHeartbeat.UnixNano() < now.Add(-config.GlobalConfiguration.NodeSchedulerHeartbeatTimeout).UnixNano() {
				nodeScheduler.ConnectionStatus = models.ConnectionStatusDisconnected
				nodeScheduler.RunningStatus = models.NodeSchedulerRunningStatusStopped
			} else {
				nodeScheduler.ConnectionStatus = models.ConnectionStatusConnected
			}

			// If status changed to disconnected, we need to rebalance
			if existing.ConnectionStatus != models.ConnectionStatusDisconnected && nodeScheduler.ConnectionStatus == models.ConnectionStatusDisconnected {
				rebalanceNeeded = true
			}

			_, err = exchangeRepo.UpdateNodeScheduler(&nodeScheduler, now)
		} else {
			nodeScheduler.ConnectionStatus = models.ConnectionStatusConnected
			_, err = exchangeRepo.CreateNodeScheduler(&nodeScheduler, now)
			// New node created, we need to rebalance
			rebalanceNeeded = true
		}

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		if rebalanceNeeded {
			balancingRepo, err := db.NewNodeSchedulerBalancingRepository(uow, idFactory)
			if err == nil {
				state, err := balancingRepo.GetState(now)
				if err == nil && state != nil && state.Status == models.Balanced {
					state.Status = models.RequestForNewBalancing
					_, _ = balancingRepo.UpsertState(state, now)
				}
			}
		}

		resultNodeSchedulers = append(resultNodeSchedulers, nodeScheduler)
	}

	commandResult.Result = resultNodeSchedulers
	return *commandResult
}
