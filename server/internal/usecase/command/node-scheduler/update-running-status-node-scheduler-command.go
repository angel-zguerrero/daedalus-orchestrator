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
	gob.Register(UpdateRunningStatusNodeSchedulerCommand{})
	gob.Register(models.NodeScheduler{})
	gob.Register([]models.NodeScheduler{})
}

type UpdateRunningStatusNodeSchedulerCommand struct {
	NodeSchedulers []models.NodeScheduler
	RunningStatus  models.NodeSchedulerRunningStatus
}

func (cmd *UpdateRunningStatusNodeSchedulerCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	exchangeRepo, err := db.NewNodeSchedulerRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var resultNodeSchedulers []models.NodeScheduler

	for _, nodeScheduler := range cmd.NodeSchedulers {
		// Look for existing exchange by code (primary upsert strategy)
		existing, err := exchangeRepo.FindByField("ID", nodeScheduler.ID, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		nodeScheduler.TTL = config.GlobalConfiguration.NodeSchedulerTTL * 60 // Convert minutes to seconds

		if existing != nil {
			nodeScheduler.ID = existing.ID
			nodeScheduler.Name = existing.Name
			nodeScheduler.CreatedAt = existing.CreatedAt
			nodeScheduler.LastHeartbeat = existing.LastHeartbeat
			nodeScheduler.ConnectionStatus = existing.ConnectionStatus
			nodeScheduler.RunningStatus = cmd.RunningStatus

			_, err = exchangeRepo.UpdateNodeScheduler(&nodeScheduler, now)
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
