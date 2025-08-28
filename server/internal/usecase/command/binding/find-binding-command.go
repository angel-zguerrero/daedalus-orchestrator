package binding

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(FindBindingCommand{})
}

type FindBindingCommand struct {
	ExchangeCode string
	QueueCode    string
	VNamespace   string
	CF           string
	CFS          string
}

func (cmd *FindBindingCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DefaultIDGeneratorFactory{}
	fmt.Println("Executing FindBindingCommand for ExchangeCode:", cmd.ExchangeCode, "QueueCode:", cmd.QueueCode, "VNamespace:", cmd.VNamespace, "CF:", cmd.CF, "CFS:", cmd.CFS)

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

	commandResult.Result = *binding
	return *commandResult
}
