package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(AssertQueueCommand{})
	gob.Register(models.Queue{})
	gob.Register([]models.Queue{})
	gob.Register(models.RoutingHeader{})
}

type AssertQueueCommand struct {
	Queues []models.Queue
	CF     string
	CFS    string
}

func (cmd *AssertQueueCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
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

	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var resultQueues []models.Queue
	newQueuesCount := 0

	for _, queue := range cmd.Queues {

		// Validate that code is not empty
		if queue.Code == "" {
			commandResult.Error = "Queue code is required"
			return *commandResult
		}

		// Validate that VNamespace is not empty
		if queue.VNamespace == "" {
			queue.VNamespace = "default"
		}

		// Validate Dead Letter Exchange if provided
		if queue.DeadLetterExchangeId != "" {
			exchange, err := exchangeRepo.GetExchangeById(queue.DeadLetterExchangeId, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}

			if exchange == nil {
				commandResult.Error = "Dead Letter Exchange with ID " + queue.DeadLetterExchangeId + " not found"
				return *commandResult
			}

			// Validate exchange type - only Direct, Topic, and Fanout are allowed
			if exchange.Type != models.Direct && exchange.Type != models.Topic && exchange.Type != models.Fanout {
				commandResult.Error = "Dead Letter Exchange must be of type Direct, Topic, or Fanout. Current type: " + string(exchange.Type)
				return *commandResult
			}

			// Validate routing key/pattern is provided for Direct and Topic exchanges
			if (exchange.Type == models.Direct || exchange.Type == models.Topic) && queue.DeadLetterExchangeRoutingKeyOrPattern == "" {
				commandResult.Error = "Dead Letter Exchange routing key/pattern is required for Direct and Topic exchanges"
				return *commandResult
			}
		}

		// Look for existing queue by code (primary upsert strategy)
		existing, err := queueRepo.GetQueueByCode(queue.Code, queue.VNamespace, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		if existing != nil {
			// Update: preserve the existing code and other immutable fields
			queue.ID = existing.ID
			queue.Code = existing.Code // Frontend cannot edit code
			queue.Type = existing.Type
			queue.VNamespace = existing.VNamespace
			queue.CreatedAt = existing.CreatedAt
			queue.PriorityThresholds = existing.PriorityThresholds

			_, err = queueRepo.UpdateQueue(&queue, now)
		} else {
			// For new queues, generate ID first if empty
			if queue.ID == "" {
				queue.ID = idFactory.GenerateID()
			}

			// Upsert VNamespace if it exists (now that we have an ID)
			if queue.VNamespace != "" {
				existingVNamespace, err := vNamespaceRepo.GetVNamespaceByName(queue.VNamespace, now)
				if err != nil {
					commandResult.Error = err.Error()
					return *commandResult
				}

				if existingVNamespace == nil {
					// Create new VNamespace
					vNamespace := models.VNamespace{
						ID:   queue.ID, // Use Queue ID as VNamespace ID
						Name: queue.VNamespace,
					}
					_, err = vNamespaceRepo.CreateVNamespace(&vNamespace, now)
					if err != nil {
						commandResult.Error = err.Error()
						return *commandResult
					}
				}
			}

			_, err = queueRepo.CreateQueue(&queue, now)

			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}

			newQueuesCount++
		}

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		// Update headers if provided
		if queue.Headers != nil && len(queue.Headers) > 0 {
			err = cmd.upsertQueueHeaders(routingHeadersRepo, queue, queue.Headers, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}
		}

		resultQueues = append(resultQueues, queue)
	}

	// Update tenant summary with the total count of new queues created
	if newQueuesCount > 0 {
		err = tenantSummaryRepo.UpdateCounters(cmd.CFS, 0, 0, newQueuesCount, 0, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	commandResult.Result = resultQueues
	return *commandResult
}

// upsertQueueHeaders creates or updates routing headers for a queue
func (cmd *AssertQueueCommand) upsertQueueHeaders(routingHeadersRepo *db.RoutingHeadersRepository, queue models.Queue, headers map[string]string, now time.Time) error {
	// Get existing headers for this queue
	existingHeaders, err := routingHeadersRepo.GetRoutingHeadersByQueue(queue.ID, now)
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
				existingHeader.HeaderType = models.HeaderTypeQueue
				_, err := routingHeadersRepo.UpdateRoutingHeader(existingHeader, now)
				if err != nil {
					return err
				}
			}
		} else {

			headerID := queue.ID + "_" + key
			// Create new header
			routingHeader := &models.RoutingHeader{
				ID:         headerID,
				QueueID:    queue.ID,
				VNamespace: queue.VNamespace,
				Key:        key,
				Value:      value,
				HeaderType: models.HeaderTypeQueue,
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
