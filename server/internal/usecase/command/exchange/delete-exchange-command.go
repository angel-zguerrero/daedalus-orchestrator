package exchange_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(DeleteExchangeCommand{})
}

type DeleteExchangeCommand struct {
	Code       string
	VNamespace string
	CF         string
	CFS        string
}

func (cmd *DeleteExchangeCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	bindingRepo, err := db.NewBindingRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	routingHeadersRepo, err := db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	tenantSummaryRepo, err := db.NewTenantSummaryRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// First find the exchange by code
	exchange, err := exchangeRepo.GetExchangeByCode(cmd.Code, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if exchange == nil {
		commandResult.Error = "exchange not found"
		return *commandResult
	}

	// Find and delete all bindings associated with this exchange (with pagination)
	bindingCount := 0
	cursor := ""

	// 1. Delete bindings where this exchange is the main exchange (ExchangeID)
	for {
		bindingsResult, err := bindingRepo.Find("ExchangeID = "+exchange.ID, 100, cursor, now)
		if err != nil {
			commandResult.Error = "error retrieving exchange bindings: " + err.Error()
			return *commandResult
		}

		if bindingsResult == nil || len(bindingsResult.Entities) == 0 {
			break
		}

		for _, binding := range bindingsResult.Entities {
			// Delete all routing headers associated with this binding
			headersResult, err := routingHeadersRepo.GetRoutingHeadersByBinding(binding.ID, now)
			if err != nil {
				commandResult.Error = "error retrieving binding headers: " + err.Error()
				return *commandResult
			}

			if headersResult != nil && len(headersResult.Entities) > 0 {
				for _, header := range headersResult.Entities {
					_, err := routingHeadersRepo.DeleteRoutingHeader(header.ID, now)
					if err != nil {
						commandResult.Error = "error deleting binding header: " + err.Error()
						return *commandResult
					}
				}
			}

			// Delete the binding
			_, err = bindingRepo.DeleteBinding(binding.ID, now)
			if err != nil {
				commandResult.Error = "error deleting binding: " + err.Error()
				return *commandResult
			}
			bindingCount++
		}

		// Update cursor for next page
		cursor = bindingsResult.Cursor
		if cursor == "" {
			break
		}
	}

	// 2. Delete bindings where this exchange is the target exchange (TargetExchangeID)
	cursor = ""
	for {
		targetBindingsResult, err := bindingRepo.Find("TargetExchangeID = "+exchange.ID, 100, cursor, now)
		if err != nil {
			commandResult.Error = "error retrieving target exchange bindings: " + err.Error()
			return *commandResult
		}

		if targetBindingsResult == nil || len(targetBindingsResult.Entities) == 0 {
			break
		}

		for _, binding := range targetBindingsResult.Entities {
			// Delete all routing headers associated with this binding
			headersResult, err := routingHeadersRepo.GetRoutingHeadersByBinding(binding.ID, now)
			if err != nil {
				commandResult.Error = "error retrieving target binding headers: " + err.Error()
				return *commandResult
			}

			if headersResult != nil && len(headersResult.Entities) > 0 {
				for _, header := range headersResult.Entities {
					_, err := routingHeadersRepo.DeleteRoutingHeader(header.ID, now)
					if err != nil {
						commandResult.Error = "error deleting target binding header: " + err.Error()
						return *commandResult
					}
				}
			}

			// Delete the binding
			_, err = bindingRepo.DeleteBinding(binding.ID, now)
			if err != nil {
				commandResult.Error = "error deleting target binding: " + err.Error()
				return *commandResult
			}
			bindingCount++
		}

		// Update cursor for next page
		cursor = targetBindingsResult.Cursor
		if cursor == "" {
			break
		}
	}

	// 3. Clear AlternateExchangeID field in bindings where this exchange is the alternate exchange
	cursor = ""
	for {
		alternateBindingsResult, err := bindingRepo.Find("AlternateExchangeID = "+exchange.ID, 100, cursor, now)
		if err != nil {
			commandResult.Error = "error retrieving alternate exchange bindings: " + err.Error()
			return *commandResult
		}

		if alternateBindingsResult == nil || len(alternateBindingsResult.Entities) == 0 {
			break
		}

		for _, binding := range alternateBindingsResult.Entities {
			// Clear the AlternateExchangeID field instead of deleting the binding
			_, err := bindingRepo.ClearAlternateExchangeId(binding.ID, now)
			if err != nil {
				commandResult.Error = "error clearing alternate exchange ID: " + err.Error()
				return *commandResult
			}
		}

		// Update cursor for next page
		cursor = alternateBindingsResult.Cursor
		if cursor == "" {
			break
		}
	}

	// Now delete the exchange by ID
	deleted, err := exchangeRepo.DeleteExchangeById(exchange.ID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if !deleted {
		commandResult.Error = "exchange not found or could not be deleted"
		return *commandResult
	}

	// Update tenant summary with a single operation
	// Decrease exchange count by 1 and binding count by bindingCount
	err = tenantSummaryRepo.UpdateCounters(cmd.CFS, 0, -1, 0, -bindingCount, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
