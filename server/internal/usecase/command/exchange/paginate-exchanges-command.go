package exchange_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateExchangesCommand{})
	gob.Register(db.FindResult[models.Exchange]{})
}

type PaginateExchangesCommand struct {
	Query      string
	Cursor     string
	PageSize   int
	VNamespace string
	CF         string
	CFS        string
}

func (cmd *PaginateExchangesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
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

	findResult, err := exchangeRepo.Paginate(cmd.Query, cmd.PageSize, cmd.Cursor, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Load headers for each exchange
	if findResult != nil && len(findResult.Entities) > 0 {
		for i := range findResult.Entities {
			headers := make(map[string]string)
			headersResult, err := routingHeadersRepo.GetRoutingHeadersByExchange(findResult.Entities[i].ID, now)
			if err == nil && headersResult != nil {
				for _, header := range headersResult.Entities {
					headers[header.Key] = header.Value
				}
			}
			findResult.Entities[i].Headers = headers
		}
	}

	commandResult.Result = *findResult
	return *commandResult
}
