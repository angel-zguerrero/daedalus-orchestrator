package exchange_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(DeleteExchangeCommand{})
}

type DeleteExchangeCommand struct {
	ID  string
	CF  string
	CFS string
}

func (cmd *DeleteExchangeCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	deleted, err := exchangeRepo.DeleteExchangeById(cmd.ID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if !deleted {
		commandResult.Error = "exchange not found or could not be deleted"
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
