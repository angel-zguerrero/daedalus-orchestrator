package exchange_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(FindExchangeByIDCommand{})
}

type FindExchangeByIDCommand struct {
	ID         string
	VNamespace string
	CF         string
	CFS        string
}

func (cmd *FindExchangeByIDCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	routingHeadersRepo, err := db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
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

	// Load headers for the exchange
	headers := make(map[string]string)
	headersResult, err := routingHeadersRepo.GetRoutingHeadersByExchange(exchange.ID, now)
	if err == nil && headersResult != nil {
		for _, header := range headersResult.Entities {
			headers[header.Key] = header.Value
		}
	}
	exchange.Headers = headers

	commandResult.Result = *exchange
	return *commandResult
}
