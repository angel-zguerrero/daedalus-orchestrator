package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(DeleteOutboxEventsCommand{})
}

type DeleteOutboxEventsCommand struct {
	EventIDs []string
	CFS      string
}

func (cmd *DeleteOutboxEventsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	outboxRepo, err := db.NewOutboxEventRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	for _, eventID := range cmd.EventIDs {
		_, err := outboxRepo.DeleteEvent(eventID, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	commandResult.Result = true
	return *commandResult
}
