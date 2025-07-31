package exchange_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(FindExchangeCommand{})
}

type FindExchangeCommand struct {
	ID  string
	CF  string
	CFS string
}

func (cmd *FindExchangeCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	fmt.Println("Executing FindExchangeCommand for ID:", cmd.ID, "CF:", cmd.CF, "CFS:", cmd.CFS)
	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	exchange, err := exchangeRepo.GetExchangeById(cmd.ID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if exchange == nil {
		commandResult.Error = "exchange not found"
		return *commandResult
	}

	commandResult.Result = *exchange
	return *commandResult
}
