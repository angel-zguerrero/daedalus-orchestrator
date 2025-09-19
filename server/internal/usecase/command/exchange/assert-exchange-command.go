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

	tenantSummaryRepo, err := db.NewTenantSummaryRepository(uow, idFactory)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	routingHeadersRepo, err := db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var resultExchanges []models.Exchange
	newExchangesCount := 0

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
		existing, err := exchangeRepo.GetExchangeByCode(exchange.Code, exchange.VNamespace, now)
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

			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}

			newExchangesCount++
		}

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		// Update headers if provided
		if exchange.Headers != nil && len(exchange.Headers) > 0 {
			err = cmd.upsertExchangeHeaders(routingHeadersRepo, exchange.ID, exchange.Headers, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}
		}

		resultExchanges = append(resultExchanges, exchange)
	}

	// Update tenant summary with the total count of new exchanges created
	if newExchangesCount > 0 {
		err = tenantSummaryRepo.UpdateCounters(cmd.CFS, 0, newExchangesCount, 0, 0, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	commandResult.Result = resultExchanges
	return *commandResult
}

// upsertExchangeHeaders creates or updates routing headers for an exchange
func (cmd *AssertExchangeCommand) upsertExchangeHeaders(routingHeadersRepo *db.RoutingHeadersRepository, exchangeID string, headers map[string]string, now time.Time) error {
	// Get existing headers for this exchange
	existingHeaders, err := routingHeadersRepo.GetRoutingHeadersByExchange(exchangeID, now)
	if err != nil {
		return err
	}

	// Create a map for quick lookup of existing headers
	existingByKey := make(map[string]*models.RoutingHeader)
	if existingHeaders != nil {
		for i := range existingHeaders.Entities {
			header := &existingHeaders.Entities[i]
			existingByKey[header.Key] = header
		}
	}

	// Process each header from input
	for key, value := range headers {
		if existingHeader, exists := existingByKey[key]; exists {
			// Update existing header if value changed
			if existingHeader.Value != value {
				existingHeader.Value = value
				_, err := routingHeadersRepo.UpdateRoutingHeader(existingHeader, now)
				if err != nil {
					return err
				}
			}
		} else {
			headerID := exchangeID + "_" + key
			// Create new header
			routingHeader := &models.RoutingHeader{
				ID:         headerID,
				ExchangeID: exchangeID,
				Key:        key,
				Value:      value,
				HeaderType: models.HeaderTypeExchange,
			}
			_, err := routingHeadersRepo.CreateRoutingHeader(routingHeader, now)
			if err != nil {
				return err
			}
		}
	}

	// Remove headers that are no longer in the input
	if existingHeaders != nil {
		for _, existingHeader := range existingHeaders.Entities {
			if _, stillExists := headers[existingHeader.Key]; !stillExists {
				_, err := routingHeadersRepo.DeleteRoutingHeader(existingHeader.ID, now)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
