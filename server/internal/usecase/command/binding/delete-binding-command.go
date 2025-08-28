package binding

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(DeleteBindingCommand{})
}

type DeleteBindingCommand struct {
	ExchangeCode string
	QueueCode    string
	VNamespace   string
	CF           string
	CFS          string
}

func (cmd *DeleteBindingCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DefaultIDGeneratorFactory{}
	
	// Get repositories
	bindingRepo, err := db.NewBindingRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

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

	tenantSummaryRepo, err := db.NewTenantSummaryRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Find Exchange by Code and VNamespace
	exchange, err := exchangeRepo.GetExchangeByCode(cmd.ExchangeCode, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	if exchange == nil {
		commandResult.Error = "exchange not found"
		return *commandResult
	}

	// Find Queue by Code and VNamespace
	queue, err := queueRepo.GetQueueByCode(cmd.QueueCode, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	if queue == nil {
		commandResult.Error = "queue not found"
		return *commandResult
	}

	// Find binding by ExchangeID and QueueID
	binding, err := bindingRepo.GetBindingByExchangeAndQueue(exchange.ID, queue.ID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	if binding == nil {
		commandResult.Error = "binding not found"
		return *commandResult
	}

	// Delete the binding by ID
	deleted, err := bindingRepo.DeleteBinding(binding.ID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if !deleted {
		commandResult.Error = "binding not found or could not be deleted"
		return *commandResult
	}

	// Update tenant summary
	err = tenantSummaryRepo.DecreaseBindingCount(cmd.CFS, 1, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
