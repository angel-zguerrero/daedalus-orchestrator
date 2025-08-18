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

		if existing != nil {
			nodeScheduler.ID = existing.ID
			nodeScheduler.Name = existing.Name
			nodeScheduler.CreatedAt = existing.CreatedAt

			if nodeScheduler.LastHeartbeat.UnixNano() < now.Add(-config.GlobalConfiguration.NodeSchedulerHeartbeatTimeout).UnixNano() {
				nodeScheduler.ConnectionStatus = models.ConnectionStatusDisconnected
			} else {
				nodeScheduler.ConnectionStatus = models.ConnectionStatusConnected

			}

			_, err = exchangeRepo.UpdateNodeScheduler(&nodeScheduler, now)
		} else {
			nodeScheduler.ConnectionStatus = models.ConnectionStatusConnected
			_, err = exchangeRepo.CreateNodeScheduler(&nodeScheduler, now)
		}

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		resultNodeSchedulers = append(resultNodeSchedulers, nodeScheduler)
	}

	commandResult.Result = resultNodeSchedulers
	return *commandResult
}
