package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(GetOutboxEventsCommand{})
	gob.Register(models.OutboxEvent{})
	gob.Register([]models.OutboxEvent{})
}

type GetOutboxEventsCommand struct {
	CFS string
}

func (cmd *GetOutboxEventsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	outboxRepo, err := db.NewOutboxEventRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	events, err := outboxRepo.GetAllEvents(now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = events
	return *commandResult
}
