package binding

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateBindingsCommand{})
	gob.Register(db.FindResult[models.Binding]{})
}

type PaginateBindingsCommand struct {
	Query          string
	Cursor         string
	PageSize       int
	VNamespace     string
	IncludeObjects bool
	CF             string
	CFS            string
}

func (cmd *PaginateBindingsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DefaultIDGeneratorFactory{}
	bindingRepo, err := db.NewBindingRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Siempre usar el método normal de paginación
	findResult, err := bindingRepo.Paginate(cmd.Query, cmd.PageSize, cmd.Cursor, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if cmd.IncludeObjects {
		// Obtener repositorios para exchanges y queues
		exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		// Obtener repositorio para routing headers
		routingHeadersRepo, err := db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		// Poblar los campos virtuales de cada binding
		for i := range findResult.Entities {
			binding := &findResult.Entities[i]

			// Obtener el exchange
			if exchange, err := exchangeRepo.GetExchangeById(binding.ExchangeID, now); err == nil && exchange != nil {
				binding.Exchange = exchange
				binding.ExchangeCode = exchange.Code

				// Si el exchange es del tipo Headers, obtener los headers del binding
				if exchange.Type == models.Headers {
					if headersResult, err := routingHeadersRepo.GetRoutingHeadersByBinding(binding.ID, now); err == nil && headersResult != nil {
						// Convertir los headers a map[string]string
						headers := make(map[string]string)
						for _, header := range headersResult.Entities {
							headers[header.Key] = header.Value
						}
						binding.Headers = headers
					}
				}
			}

			// Obtener la queue (solo para bindings classic)
			if binding.QueueID != "" {
				if queue, err := queueRepo.GetQueueById(binding.QueueID, now); err == nil && queue != nil {
					binding.Queue = queue
					binding.QueueCode = queue.Code
				}
			}
		}
	}

	commandResult.Result = *findResult
	return *commandResult
}
