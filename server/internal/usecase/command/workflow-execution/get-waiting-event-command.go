package workflow_execution

import (
	"encoding/gob"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
)

func init() {
	gob.Register(GetWaitingEventCommand{})
	gob.Register(models.WaitingEvent{})
	gob.Register([]models.WaitingEvent{})
}

type GetWaitingEventCommand struct {
	ID          string
	ExecutionID string
	CF          string
	CFS         string
}

func (cmd *GetWaitingEventCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	idFactory := &db.DeterministicIDGeneratorFactory{}

	repo, err := db.NewWaitingEventRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if cmd.ID != "" {
		evt, err := repo.GetWaitingEventByID(cmd.ID, now)
		if err != nil || evt == nil {
			commandResult.Error = fmt.Sprintf("waiting event with ID %s not found", cmd.ID)
			return *commandResult
		}
		commandResult.Result = *evt
		return *commandResult
	}

	if cmd.ExecutionID != "" {
		events, err := repo.GetWaitingEventsByExecutionID(cmd.ExecutionID, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		commandResult.Result = events
		return *commandResult
	}

	commandResult.Error = "ID or ExecutionID is required"
	return *commandResult
}
