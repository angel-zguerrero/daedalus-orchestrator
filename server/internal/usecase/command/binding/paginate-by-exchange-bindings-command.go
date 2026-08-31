package binding

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateByExchangeBindingsCommand{})
	gob.Register(db.FindResult[models.Binding]{})
}

type PaginateByExchangeBindingsCommand struct {
	ExchangeID     string
	Cursor         string
	PageSize       int
	VNamespace     string
	IncludeObjects bool
	CF             string
	CFS            string
	// UseGroupIndex if true, uses the group index for O(1) lookup of all bindings by ExchangeID.
	// This bypasses pagination and returns all bindings in a single read.
	UseGroupIndex bool
}

func (cmd *PaginateByExchangeBindingsCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DefaultIDGeneratorFactory{}
	bindingRepo, err := db.NewBindingRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var entities []models.Binding
	var cursor string

	if cmd.UseGroupIndex {
		// O(1) group index lookup - single KV read returns all IDs, then bulk data fetch
		entities, err = bindingRepo.GetBindingsByExchangeID(cmd.ExchangeID, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		cursor = "" // Group index returns all at once, no pagination needed
	} else {
		// Siempre usar el método normal de paginación
		findResult, err := bindingRepo.PaginateByExchangeID(cmd.ExchangeID, cmd.PageSize, cmd.Cursor, cmd.VNamespace, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		entities = findResult.Entities
		cursor = findResult.Cursor
	}

	result := db.FindResult[models.Binding]{
		Entities: entities,
		Cursor:   cursor,
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
		for i := range result.Entities {
			binding := &result.Entities[i]

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

			// Obtener el target exchange
			if binding.TargetExchangeID != "" {
				if targetExchange, err := exchangeRepo.GetExchangeById(binding.TargetExchangeID, now); err == nil && targetExchange != nil {
					binding.TargetExchange = targetExchange
					binding.TargetExchangeCode = targetExchange.Code
				}
			}

			// Obtener el alternate exchange
			if binding.AlternateExchangeID != "" {
				if alternateExchange, err := exchangeRepo.GetExchangeById(binding.AlternateExchangeID, now); err == nil && alternateExchange != nil {
					binding.AlternateExchange = alternateExchange
					binding.AlternateExchangeCode = alternateExchange.Code
				}
			}
		}
	}

	commandResult.Result = result
	return *commandResult
}
