package exchange_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(AssertExchangeCommand{})
	gob.Register(models.Exchange{})
	gob.Register([]models.Exchange{})
}

type AssertExchangeCommand struct {
	Exchanges []models.Exchange
	CF        string
	CFS       string
}

func (cmd *AssertExchangeCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	vNamespaceRepo, err := db.NewVNamespaceRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var resultExchanges []models.Exchange

	for _, exchange := range cmd.Exchanges {

		// Upsert VNamespace if it exists
		if exchange.VNamespace != "" {
			existingVNamespace, err := vNamespaceRepo.GetVNamespaceByName(exchange.VNamespace, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}

			if existingVNamespace == nil {
				// Create new VNamespace
				vNamespace := models.VNamespace{
					ID:   exchange.ID, // Use Exchange ID as VNamespace ID
					Name: exchange.VNamespace,
				}
				_, err = vNamespaceRepo.CreateVNamespace(&vNamespace, now)
				if err != nil {
					commandResult.Error = err.Error()
					return *commandResult
				}
			}
		}

		existing, err := exchangeRepo.GetExchangeByName(exchange.Name, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		if existing != nil {
			exchange.ID = existing.ID
			exchange.CreatedAt = existing.CreatedAt
			exchange.VNamespace = existing.VNamespace
			exchange.Type = existing.Type
			_, err = exchangeRepo.UpdateExchange(&exchange, now)
		} else {
			_, err = exchangeRepo.CreateExchange(&exchange, now)
		}

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		resultExchanges = append(resultExchanges, exchange)
	}

	commandResult.Result = resultExchanges
	return *commandResult
}
