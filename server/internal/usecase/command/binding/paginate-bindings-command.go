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
	gob.Register(db.FindResult[models.BindingWithObjects]{})
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
		// Convertir a BindingWithObjects y agregar los objetos Exchange y Queue
		resultWithObjects := &db.FindResult[models.BindingWithObjects]{
			Cursor:   findResult.Cursor,
			Entities: make([]models.BindingWithObjects, 0, len(findResult.Entities)),
		}

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

		// Convertir cada binding y añadir los objetos
		for _, binding := range findResult.Entities {
			bindingWithObjects := models.BindingWithObjects{
				ID:          binding.ID,
				VNamespace:  binding.VNamespace,
				ExchangeID:  binding.ExchangeID,
				QueueID:     binding.QueueID,
				RoutingKey:  binding.RoutingKey,
				Pattern:     binding.Pattern,
				XMatch:      binding.XMatch,
				BindingType: binding.BindingType,
				CreatedAt:   binding.CreatedAt,
				UpdatedAt:   binding.UpdatedAt,
			}

			// Obtener el exchange
			if exchange, err := exchangeRepo.GetExchangeById(binding.ExchangeID, now); err == nil && exchange != nil {
				bindingWithObjects.Exchange = exchange
				bindingWithObjects.ExchangeCode = exchange.Code
			}

			// Obtener la queue
			if queue, err := queueRepo.GetQueueById(binding.QueueID, now); err == nil && queue != nil {
				bindingWithObjects.Queue = queue
				bindingWithObjects.QueueCode = queue.Code
			}

			resultWithObjects.Entities = append(resultWithObjects.Entities, bindingWithObjects)
		}

		commandResult.Result = *resultWithObjects
	} else {
		commandResult.Result = *findResult
	}

	return *commandResult
}
