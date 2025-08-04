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

		// Validate that code is not empty
		if exchange.Code == "" {
			commandResult.Error = "Exchange code is required"
			return *commandResult
		}

		// Validate that VNamespace is not empty
		if exchange.VNamespace == "" {
			exchange.VNamespace = "default"
		}

		// Look for existing exchange by code (primary upsert strategy)
		existing, err := exchangeRepo.GetExchangeByCode(exchange.Code, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		if existing != nil {
			// Update: preserve the existing code and other immutable fields
			exchange.ID = existing.ID
			exchange.Code = existing.Code // Frontend cannot edit code
			exchange.Type = existing.Type
			exchange.VNamespace = existing.VNamespace
			exchange.CreatedAt = existing.CreatedAt

			_, err = exchangeRepo.UpdateExchange(&exchange, now)
		} else {
			// For new exchanges, generate ID first if empty
			if exchange.ID == "" {
				exchange.ID = idFactory.GenerateID()
			}

			// Upsert VNamespace if it exists (now that we have an ID)
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
