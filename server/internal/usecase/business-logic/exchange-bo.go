package business_logic

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"fmt"

	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	binding_command "deadalus-orch/server/internal/usecase/command/binding"
	exchange_command "deadalus-orch/server/internal/usecase/command/exchange"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	header_command "deadalus-orch/server/internal/usecase/command/header"
	"deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ExchangeBO struct {
	Config *common.ServerConfing
}

func NewExchangeBO(Config *common.ServerConfing) *ExchangeBO {
	return &ExchangeBO{
		Config: Config,
	}
}

func (bo *ExchangeBO) CreateExchange(ctx context.Context, code, vnamespace, name string, exchangeType models.ExchangeType, headers map[string]string, cf, cfs string) (models.Exchange, error) {
	exchange := &models.Exchange{
		ID:         strings.ReplaceAll(uuid.New().String(), "-", ""),
		Code:       code,
		Name:       name,
		Type:       exchangeType,
		VNamespace: vnamespace,
		Headers:    headers,
	}

	createdList, err := bo.BulkCreateExchange(ctx, []*models.Exchange{exchange}, cf, cfs)
	if err != nil {
		return models.Exchange{}, err
	}
	return createdList[0], nil
}

func (bo *ExchangeBO) BulkCreateExchange(ctx context.Context, exchanges []*models.Exchange, cf, cfs string) ([]models.Exchange, error) {
	if len(exchanges) == 0 {
		return nil, errors.New("no exchanges provided")
	}

	// Asegurar IDs válidos
	for _, t := range exchanges {
		if t.ID == "" {
			t.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
		}
	}

	asseertExchangeCommand := &exchange_command.AssertExchangeCommand{
		Exchanges: make([]models.Exchange, len(exchanges)),
		CF:        cf,
		CFS:       cfs,
	}
	for i, t := range exchanges {
		asseertExchangeCommand.Exchanges[i] = *t
	}

	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout*time.Duration(len(exchanges)))
	defer writeCancel()

	fsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  asseertExchangeCommand,
	}

	result, err := bo.Config.TenantNodesDictionary[cfs].Write(writeCtx, fsmCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Failed to assert exchanges (bulk)")
		return nil, fmt.Errorf("failed to assert exchanges (bulk): %w", err)
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Bulk exchange creation command returned unexpected result type")
		return nil, fmt.Errorf("bulk exchange creation command returned decode error: %w", err)
	}

	if parsedResult.Error != "" {
		return nil, fmt.Errorf("bulk exchange creation failed: %s", parsedResult.Error)
	}

	created := parsedResult.Result.([]models.Exchange)

	return created, nil
}

func (bo *ExchangeBO) GetExchange(ctx context.Context, exchangeCode, vnamespace, cf, cfs string) (models.Exchange, error) {
	findExchangeCommand := &exchange_command.FindExchangeCommand{
		Code:       exchangeCode,
		VNamespace: vnamespace,
		CF:         cf,
		CFS:        cfs,
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: findExchangeCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.TenantNodesDictionary[cfs].Read(readCtx, *queryCommand)
	if err != nil {
		if strings.Contains(err.Error(), "cannot encode nil pointer of type") {
			return models.Exchange{}, errors.New("Exchange not found")
		}
		bo.Config.Logger.Error().Err(err).Msg("Find exchange command failed")
		return models.Exchange{}, errors.New("Find exchange command failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Find exchange command failed")
		return models.Exchange{}, errors.New("Find exchange command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find exchange command failed")
		return models.Exchange{}, errors.New("Find exchange command failed")
	}

	if parsedResult.Result == nil {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find exchange command failed")
		return models.Exchange{}, errors.New("Exchange not found")
	}

	exchange := parsedResult.Result.(models.Exchange)

	// Para exchanges globales no hay nodo específico
	return exchange, nil
}

func (bo *ExchangeBO) DeleteExchange(ctx context.Context, exchangeCode, vnamespace, cf, cfs string) error {
	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer writeCancel()

	deleteExchangeCommand := &exchange_command.DeleteExchangeCommand{
		Code:       exchangeCode,
		VNamespace: vnamespace,
		CF:         cf,
		CFS:        cfs,
	}

	atstCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  deleteExchangeCommand,
	}

	result, err := bo.Config.TenantNodesDictionary[cfs].Write(writeCtx, atstCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Str("ExchangeCode", exchangeCode).Str("VNamespace", vnamespace).Msg("Failed to delete exchange")
		return errors.New("Failed to delete exchange: " + err.Error())
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Str("ExchangeCode", exchangeCode).Str("VNamespace", vnamespace).Msg("Exchange deletion command returned unexpected result type")
		return errors.New("Exchange deletion command returned unexpected error")
	}

	if parsedResult.Error != "" {
		return errors.New("Failed to delete exchange error: " + parsedResult.Error)
	}

	bo.Config.Logger.Info().Str("ExchangeCode", exchangeCode).Str("VNamespace", vnamespace).Msg("exchange deleted successfully")
	return nil
}

func (bo *ExchangeBO) GetExchanges(ctx context.Context, q string, cursor string, pageSize int, vNamespace string, cf, cfs string) (db.FindResult[models.Exchange], error) {
	paginateExchangesCommand := &exchange_command.PaginateExchangesCommand{
		Query:      q,
		Cursor:     cursor,
		PageSize:   pageSize,
		VNamespace: vNamespace,
		CF:         cf,
		CFS:        cfs,
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: paginateExchangesCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.TenantNodesDictionary[cfs].Read(readCtx, *queryCommand)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate exchanges command failed")
		return db.FindResult[models.Exchange]{}, errors.New("Paginate exchanges failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate exchanges command failed")
		return db.FindResult[models.Exchange]{}, errors.New("Paginate exchanges command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Paginate exchanges command failed")
		return db.FindResult[models.Exchange]{}, errors.New("Paginate exchanges command failed")
	}

	findResult := parsedResult.Result.(db.FindResult[models.Exchange])
	if findResult.Entities == nil {
		findResult.Entities = []models.Exchange{}
	}

	return findResult, nil
}

func (bo *ExchangeBO) PublishMessage(ctx context.Context, exchangeCode, routingKeyOrPatternOrQueueCode string, message models.QueueMessage, vnamespace string, cf, cfs string) (map[string]string, error) {

	if message.MessageID == "" {
		message.MessageID = strings.ReplaceAll(uuid.New().String(), "-", "")
	}

	queues, err := bo.GetQueuesFromExchange(ctx, exchangeCode, routingKeyOrPatternOrQueueCode, message, vnamespace, cf, cfs)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Failed to get queues from exchange")
		return nil, fmt.Errorf("failed to get queues from exchange: %w", err)
	}

	if len(queues) == 0 {
		bo.Config.Logger.Info().Str("exchangeCode", exchangeCode).Str("routingKeyOrPatternOrQueueCode", routingKeyOrPatternOrQueueCode).Msg("No queues matched for the given routing key or pattern")
		return nil, nil
	}

	queueCodeMap := make(map[string]string, len(queues))
	for _, q := range queues {
		queueCodeMap[q.ID] = q.Code
	}

	queueMessages := make([]models.QueueMessage, len(queues))
	for i, q := range queues {
		message := models.QueueMessage{
			ID:          strings.ReplaceAll(uuid.New().String(), "-", ""),
			MessageID:   message.MessageID,
			Content:     message.Content,
			ContentType: message.ContentType,
			Headers:     message.Headers,
			QueueID:     q.ID,
			Priority:    message.Priority,
			Handler:     message.Handler,
			Parameters:  message.Parameters,
			VNamespace:  vnamespace,
		}
		queueMessages[i] = message
	}

	enqueueCommand := &queue.EnqueueCommand{
		Messages: make([]models.QueueMessage, len(queueMessages)),
		CF:       cf,
		CFS:      cfs,
	}
	copy(enqueueCommand.Messages, queueMessages)

	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout*time.Duration(len(queueMessages)))
	defer writeCancel()

	fsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  enqueueCommand,
	}

	result, err := bo.Config.TenantNodesDictionary[cfs].Write(writeCtx, fsmCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Failed to enqueue messages to queues")
		return nil, fmt.Errorf("failed to enqueue messages to queues: %w", err)
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Enqueue command returned unexpected result type")
		return nil, fmt.Errorf("enqueue command returned decode error: %w", err)
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(errors.New(parsedResult.Error)).Msg("Enqueue command failed")
		return nil, fmt.Errorf("enqueue command failed: %s", parsedResult.Error)
	}

	createdMessages := parsedResult.Result.([]models.QueueMessage)
	resultingMessages := make(map[string]string, len(createdMessages))
	for _, msg := range createdMessages {
		queueCode := queueCodeMap[msg.QueueID]
		resultingMessages[queueCode] = msg.ID
	}

	return resultingMessages, nil
}

func (bo *ExchangeBO) GetQueuesFromExchange(ctx context.Context, exchangeCode, routingKeyOrPatternOrQueueCode string, message models.QueueMessage, vnamespace string, cf, cfs string) ([]models.Queue, error) {
	// First, get the exchange
	exchange, err := bo.GetExchange(ctx, exchangeCode, vnamespace, cf, cfs)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange: %w", err)
	}

	// Get bindings for this exchange
	bindings, err := bo.getBindingsByExchange(ctx, exchange.ID, vnamespace, cf, cfs)
	if err != nil {
		return nil, fmt.Errorf("failed to get bindings for exchange: %w", err)
	}

	var resultQueues []models.Queue
	visitedExchanges := make(map[string]bool) // To prevent infinite recursion

	// Process each binding
	for _, binding := range bindings {
		queues, err := bo.processBinding(ctx, binding, routingKeyOrPatternOrQueueCode, message, cf, cfs, visitedExchanges)
		if err != nil {
			return nil, fmt.Errorf("failed to process binding: %w", err)
		}
		resultQueues = append(resultQueues, queues...)
	}

	return bo.deduplicateQueues(resultQueues), nil
}

// Helper method to get bindings by exchange ID
func (bo *ExchangeBO) getBindingsByExchange(ctx context.Context, exchangeID, vnamespace, cf, cfs string) ([]models.Binding, error) {
	var allBindings []models.Binding
	cursor := ""
	pageSize := 100

	for {
		paginateBindingsCommand := &binding_command.PaginateByExchangeBindingsCommand{
			ExchangeID:     exchangeID,
			Cursor:         cursor,
			PageSize:       pageSize,
			VNamespace:     vnamespace, // All namespaces
			IncludeObjects: true,       // Include exchange, queue objects
			CF:             cf,
			CFS:            cfs,
		}

		queryCommand := &general_command.Query_Command{
			Command: &general_command.Repository_Command{
				CMD: paginateBindingsCommand,
			},
			Now: time.Now().UnixNano(),
		}

		readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
		result, err := bo.Config.TenantNodesDictionary[cfs].Read(readCtx, *queryCommand)
		cancel()

		if err != nil {
			return nil, fmt.Errorf("failed to get bindings: %w", err)
		}

		buf := bytes.NewBuffer(result.([]byte))
		dec := gob.NewDecoder(buf)
		parsedResult := &commands.CommandResult{}
		if err := dec.Decode(parsedResult); err != nil {
			return nil, fmt.Errorf("failed to decode bindings result: %w", err)
		}

		if parsedResult.Error != "" {
			return nil, fmt.Errorf("bindings query failed: %s", parsedResult.Error)
		}

		findResult := parsedResult.Result.(db.FindResult[models.Binding])

		// Filter bindings for this specific exchange and add to collection
		allBindings = append(allBindings, findResult.Entities...)

		// Check if there are more pages
		if findResult.Cursor == "" || len(findResult.Entities) < pageSize {
			break
		}
		cursor = findResult.Cursor
	}

	return allBindings, nil
}

// Process a single binding to determine if it matches and return queues
func (bo *ExchangeBO) processBinding(ctx context.Context, binding models.Binding, routingKeyOrPatternOrQueueCode string, message models.QueueMessage, cf, cfs string, visitedExchanges map[string]bool) ([]models.Queue, error) {
	var resultQueues []models.Queue

	// Prevent infinite recursion
	if visitedExchanges[binding.ExchangeID] {
		return resultQueues, nil
	}

	switch binding.BindingType {
	case models.BindingTypeClassic:
		return bo.processClassicBinding(ctx, binding, routingKeyOrPatternOrQueueCode, message, cf, cfs, visitedExchanges)
	case models.BindingTypeDynamic:
		return bo.processDynamicBinding(ctx, binding, routingKeyOrPatternOrQueueCode, message, cf, cfs, visitedExchanges)
	default:
		return resultQueues, fmt.Errorf("unknown binding type: %s", binding.BindingType)
	}
}

// Process classic binding (static routing)
func (bo *ExchangeBO) processClassicBinding(ctx context.Context, binding models.Binding, routingKeyOrPatternOrQueueCode string, message models.QueueMessage, cf, cfs string, visitedExchanges map[string]bool) ([]models.Queue, error) {
	var resultQueues []models.Queue

	// Check if binding matches routing criteria
	if !bo.matchesRoutingCriteria(binding, routingKeyOrPatternOrQueueCode, message) {
		if binding.AlternateExchange != nil {
			// Check if we've already visited this alternate exchange to prevent infinite recursion
			if !visitedExchanges[binding.AlternateExchange.ID] {
				// Mark this alternate exchange as visited
				visitedExchanges[binding.AlternateExchange.ID] = true

				queues, err := bo.GetQueuesFromExchange(ctx, binding.AlternateExchange.Code, routingKeyOrPatternOrQueueCode, message, binding.VNamespace, cf, cfs)
				if err != nil {
					return resultQueues, fmt.Errorf("failed to get queues from alternate exchange: %w", err)
				} else {
					resultQueues = append(resultQueues, queues...)
				}
			}
		}
		return resultQueues, nil
	}

	// If target is a queue, add it directly
	if binding.TargetExchangeType == models.TargetExchangeTypeQueue && binding.Queue != nil {
		resultQueues = append(resultQueues, *binding.Queue)
	}

	// If target is an exchange, recurse
	if binding.TargetExchangeType == models.TargetExchangeTypeExchange && binding.TargetExchange != nil {
		// Check if we've already visited this target exchange to prevent infinite recursion
		if !visitedExchanges[binding.TargetExchange.ID] {
			// Mark this target exchange as visited
			visitedExchanges[binding.TargetExchange.ID] = true

			queues, err := bo.GetQueuesFromExchange(ctx, binding.TargetExchange.Code, routingKeyOrPatternOrQueueCode, message, binding.VNamespace, cf, cfs)
			if err != nil {
				return resultQueues, fmt.Errorf("failed to get queues from target exchange: %w", err)
			} else {
				resultQueues = append(resultQueues, queues...)
			}
		}
	}

	return resultQueues, nil
}

// Process dynamic binding (pattern-based routing)
func (bo *ExchangeBO) processDynamicBinding(ctx context.Context, binding models.Binding, routingKeyOrPatternOrQueueCode string, message models.QueueMessage, cf, cfs string, visitedExchanges map[string]bool) ([]models.Queue, error) {
	var resultQueues []models.Queue

	// Prevent infinite recursion by checking if we've already visited this exchange
	if binding.Exchange != nil && visitedExchanges[binding.Exchange.ID] {
		return resultQueues, nil
	}

	// Mark this exchange as visited
	if binding.Exchange != nil {
		visitedExchanges[binding.Exchange.ID] = true
	}

	if binding.TargetExchangeType == models.TargetExchangeTypeQueue {
		// Only apply this logic for direct exchanges
		if binding.Exchange != nil && binding.Exchange.Type == models.Direct {
			// Search for queue by Code using routingKeyOrPatternOrQueueCode
			findQueueCommand := &queue.FindQueueCommand{
				Code:           routingKeyOrPatternOrQueueCode,
				VNamespace:     binding.VNamespace,
				IncludeHeaders: false, // Not necessary to include headers
				CF:             cf,
				CFS:            cfs,
			}

			queryCommand := &general_command.Query_Command{
				Command: &general_command.Repository_Command{
					CMD: findQueueCommand,
				},
				Now: time.Now().UnixNano(),
			}

			readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
			defer cancel()

			result, err := bo.Config.TenantNodesDictionary[cfs].Read(readCtx, *queryCommand)
			if err != nil {
				return resultQueues, fmt.Errorf("failed to execute find queue command: %w", err)
			}

			buf := bytes.NewBuffer(result.([]byte))
			dec := gob.NewDecoder(buf)
			parsedResult := &commands.CommandResult{}
			if err := dec.Decode(parsedResult); err != nil {
				return resultQueues, fmt.Errorf("failed to decode cluster response: %w", err)
			}

			if parsedResult.Error != "" {
				return nil, fmt.Errorf("find queue command failed: %s", parsedResult.Error)
			}

			if parsedResult.Result != nil {
				foundQueue, ok := parsedResult.Result.(models.Queue)
				if ok {
					resultQueues = append(resultQueues, foundQueue)
				}
			}
		}

		// TODO: Implement logic for headers exchange type when TargetExchangeType is Queue
		if binding.Exchange != nil && binding.Exchange.Type == models.Headers {
			messageHeaders := message.Headers
			var listQueueHeaders []models.RoutingHeader
			for key := range messageHeaders {
				// Use ListHeadersCommand to get routing headers for this key
				listHeadersCommand := &header_command.ListHeadersCommand{
					Key:               key,
					RoutingHeaderType: models.HeaderTypeQueue,
					VNamespace:        binding.VNamespace,
					CF:                cf,
					CFS:               cfs,
				}

				queryCommand := &general_command.Query_Command{
					Command: &general_command.Repository_Command{
						CMD: listHeadersCommand,
					},
					Now: time.Now().UnixNano(),
				}

				readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
				result, err := bo.Config.TenantNodesDictionary[cfs].Read(readCtx, *queryCommand)
				cancel()

				if err != nil {
					return resultQueues, fmt.Errorf("failed to get routing headers for key %s: %w", key, err)
				}

				buf := bytes.NewBuffer(result.([]byte))
				dec := gob.NewDecoder(buf)
				parsedResult := &commands.CommandResult{}
				if err := dec.Decode(parsedResult); err != nil {
					return resultQueues, fmt.Errorf("failed to decode routing headers result for key %s: %w", key, err)
				}

				if parsedResult.Error != "" {
					return resultQueues, fmt.Errorf("routing headers query failed for key %s: %s", key, parsedResult.Error)
				}

				if parsedResult.Result != nil {
					allQueueHeaders := parsedResult.Result.([]models.RoutingHeader)
					listQueueHeaders = append(listQueueHeaders, allQueueHeaders...)
				}
			}

			// Cross information with binding.XMatch to get queue IDs
			var matchingQueueIDs []string

			switch binding.XMatch {
			case models.XMatchTypeAll:
				// All message headers must match - find queues that have ALL message headers with matching values
				messageHeadersCount := len(messageHeaders)
				if messageHeadersCount == 0 {
					break
				}

				// Group headers by QueueID
				queueHeadersMap := make(map[string]map[string]string)
				for _, header := range listQueueHeaders {
					if header.QueueID != "" {
						if queueHeadersMap[header.QueueID] == nil {
							queueHeadersMap[header.QueueID] = make(map[string]string)
						}
						queueHeadersMap[header.QueueID][header.Key] = header.Value
					}
				}

				// Check each queue to see if it has all message headers with matching values
				for queueID, queueHeaders := range queueHeadersMap {
					matchCount := 0
					for messageKey, messageValue := range messageHeaders {
						if queueValue, queueHasKey := queueHeaders[messageKey]; queueHasKey {
							if queueValue == messageValue {
								matchCount++
							}
						}
					}
					// Queue matches if it has all message headers with correct values
					if matchCount == messageHeadersCount {
						matchingQueueIDs = append(matchingQueueIDs, queueID)
					}
				}

			case models.XMatchTypeAny:
				// At least one message header must match - find queues that have ANY message header with matching value
				queueMatches := make(map[string]bool)
				for _, queueHeader := range listQueueHeaders {
					if queueHeader.QueueID != "" {
						// Check if this queue header matches any message header
						if messageValue, exists := messageHeaders[queueHeader.Key]; exists {
							if queueHeader.Value == messageValue {
								queueMatches[queueHeader.QueueID] = true
							}
						}
					}
				}
				// Collect all matching queue IDs
				for queueID := range queueMatches {
					matchingQueueIDs = append(matchingQueueIDs, queueID)
				}

			default:
				return nil, fmt.Errorf("unknown XMatch type: %s", binding.XMatch)
			}

			// Use FindQueueByIDsCommand to get actual Queue objects from the matching IDs
			if len(matchingQueueIDs) > 0 {
				findQueuesByIDsCommand := &queue.FindQueueByIDsCommand{
					IDs:            matchingQueueIDs,
					VNamespace:     binding.VNamespace,
					IncludeHeaders: true,
					CF:             cf,
					CFS:            cfs,
				}

				queueQueryCommand := &general_command.Query_Command{
					Command: &general_command.Repository_Command{
						CMD: findQueuesByIDsCommand,
					},
					Now: time.Now().UnixNano(),
				}

				queueReadCtx, queueCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
				queueResult, err := bo.Config.TenantNodesDictionary[cfs].Read(queueReadCtx, *queueQueryCommand)
				queueCancel()

				if err != nil {
					return resultQueues, fmt.Errorf("failed to find queues by IDs: %w", err)
				}

				queueBuf := bytes.NewBuffer(queueResult.([]byte))
				queueDec := gob.NewDecoder(queueBuf)
				queueParsedResult := &commands.CommandResult{}
				if err := queueDec.Decode(queueParsedResult); err != nil {
					return resultQueues, fmt.Errorf("failed to decode queues result: %w", err)
				}

				if queueParsedResult.Error != "" {
					return resultQueues, fmt.Errorf("queues query failed: %s", queueParsedResult.Error)
				}

				if queueParsedResult.Result != nil {
					foundQueues, ok := queueParsedResult.Result.([]models.Queue)
					if ok {
						// Si XMatch=All, filtrar los queues para que todos los headers del mensaje hagan match exacto con los headers del queue
						if binding.XMatch == models.XMatchTypeAll {
							filteredQueues := make([]models.Queue, 0, len(foundQueues))
							for _, q := range foundQueues {
								queueHeaders := q.Headers // map[string]string
								allMatch := true
								for qhKey, qhValue := range queueHeaders {
									messageValue, exists := messageHeaders[qhKey]
									if !exists || messageValue != qhValue {
										allMatch = false
										break
									}
								}
								if allMatch {
									filteredQueues = append(filteredQueues, q)
								}
							}
							resultQueues = append(resultQueues, filteredQueues...)
						} else {
							// Si XMatch=Any o cualquier otro, incluir todos los queues encontrados
							resultQueues = append(resultQueues, foundQueues...)
						}
					} else {
						return resultQueues, fmt.Errorf("unexpected result type for queues query")
					}
				}
			}

		}
	}

	if binding.TargetExchangeType == models.TargetExchangeTypeExchange {
		// Use headers to find matching exchanges and call GetQueuesFromExchange for each
		messageHeaders := message.Headers
		var listExchangeHeaders []models.RoutingHeader

		for key := range messageHeaders {
			// Use ListHeadersCommand to get routing headers for exchanges with this key
			listHeadersCommand := &header_command.ListHeadersCommand{
				Key:               key,
				RoutingHeaderType: models.HeaderTypeExchange, // Looking for exchange headers
				VNamespace:        binding.VNamespace,
				CF:                cf,
				CFS:               cfs,
			}

			queryCommand := &general_command.Query_Command{
				Command: &general_command.Repository_Command{
					CMD: listHeadersCommand,
				},
				Now: time.Now().UnixNano(),
			}

			readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
			result, err := bo.Config.TenantNodesDictionary[cfs].Read(readCtx, *queryCommand)
			cancel()

			if err != nil {
				return resultQueues, fmt.Errorf("failed to get exchange routing headers for key %s: %w", key, err)
			}

			buf := bytes.NewBuffer(result.([]byte))
			dec := gob.NewDecoder(buf)
			parsedResult := &commands.CommandResult{}
			if err := dec.Decode(parsedResult); err != nil {
				return resultQueues, fmt.Errorf("failed to decode exchange routing headers result for key %s: %w", key, err)
			}

			if parsedResult.Error != "" {
				return resultQueues, fmt.Errorf("exchange routing headers query failed for key %s: %s", key, parsedResult.Error)
			}

			if parsedResult.Result != nil {
				allExchangeHeaders := parsedResult.Result.([]models.RoutingHeader)
				listExchangeHeaders = append(listExchangeHeaders, allExchangeHeaders...)
			}
		}

		// Cross information with binding.XMatch to get exchange IDs
		var matchingExchangeIDs []string

		switch binding.XMatch {
		case models.XMatchTypeAll:
			// All message headers must match - find exchanges that have ALL message headers with matching values
			messageHeadersCount := len(messageHeaders)
			if messageHeadersCount == 0 {
				break
			}

			// Group headers by ExchangeID
			exchangeHeadersMap := make(map[string]map[string]string)
			for _, header := range listExchangeHeaders {
				if header.ExchangeID != "" {
					if exchangeHeadersMap[header.ExchangeID] == nil {
						exchangeHeadersMap[header.ExchangeID] = make(map[string]string)
					}
					exchangeHeadersMap[header.ExchangeID][header.Key] = header.Value
				}
			}

			// Check each exchange to see if it has all message headers with matching values
			for exchangeID, exchangeHeaders := range exchangeHeadersMap {
				matchCount := 0
				for messageKey, messageValue := range messageHeaders {
					if exchangeValue, exchangeHasKey := exchangeHeaders[messageKey]; exchangeHasKey {
						if exchangeValue == messageValue {
							matchCount++
						}
					}
				}
				// Exchange matches if it has all message headers with correct values
				if matchCount == messageHeadersCount {
					matchingExchangeIDs = append(matchingExchangeIDs, exchangeID)
				}
			}

		case models.XMatchTypeAny:
			// At least one message header must match - find exchanges that have ANY message header with matching value
			exchangeMatches := make(map[string]bool)
			for _, exchangeHeader := range listExchangeHeaders {
				if exchangeHeader.ExchangeID != "" {
					// Check if this exchange header matches any message header
					if messageValue, exists := messageHeaders[exchangeHeader.Key]; exists {
						if exchangeHeader.Value == messageValue {
							exchangeMatches[exchangeHeader.ExchangeID] = true
						}
					}
				}
			}
			// Collect all matching exchange IDs
			for exchangeID := range exchangeMatches {
				matchingExchangeIDs = append(matchingExchangeIDs, exchangeID)
			}

		default:
			// Unknown XMatch type, return empty
			return nil, fmt.Errorf("unknown XMatch type: %s", binding.XMatch)
		}

		// For each matching exchange, call GetQueuesFromExchange
		for _, exchangeID := range matchingExchangeIDs {
			// Skip if we've already visited this exchange to prevent infinite recursion
			if visitedExchanges[exchangeID] {
				continue
			}

			// Get the exchange by ID to get its Code
			findExchangeByIDCommand := &exchange_command.FindExchangeByIDCommand{
				ID:         exchangeID,
				VNamespace: binding.VNamespace,
				CF:         cf,
				CFS:        cfs,
			}

			exchangeQueryCommand := &general_command.Query_Command{
				Command: &general_command.Repository_Command{
					CMD: findExchangeByIDCommand,
				},
				Now: time.Now().UnixNano(),
			}

			exchangeReadCtx, exchangeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
			exchangeResult, err := bo.Config.TenantNodesDictionary[cfs].Read(exchangeReadCtx, *exchangeQueryCommand)
			exchangeCancel()

			if err != nil {
				return resultQueues, fmt.Errorf("failed to find exchange by ID %s: %w", exchangeID, err)
			}

			exchangeBuf := bytes.NewBuffer(exchangeResult.([]byte))
			exchangeDec := gob.NewDecoder(exchangeBuf)
			exchangeParsedResult := &commands.CommandResult{}
			if err := exchangeDec.Decode(exchangeParsedResult); err != nil {
				return resultQueues, fmt.Errorf("failed to decode exchange result for ID %s: %w", exchangeID, err)
			}

			if exchangeParsedResult.Error != "" {
				return resultQueues, fmt.Errorf("exchange query failed for ID %s: %s", exchangeID, exchangeParsedResult.Error)
			}

			if exchangeParsedResult.Result != nil {
				foundExchange, ok := exchangeParsedResult.Result.(models.Exchange)
				if ok {
					// Mark this exchange as visited
					visitedExchanges[exchangeID] = true

					// Call GetQueuesFromExchange for this exchange
					queues, err := bo.GetQueuesFromExchange(ctx, foundExchange.Code, routingKeyOrPatternOrQueueCode, message, binding.VNamespace, cf, cfs)
					if err != nil {
						return resultQueues, fmt.Errorf("failed to get queues from matched exchange %s: %w", foundExchange.Code, err)
					}
					resultQueues = append(resultQueues, queues...)
				}
			}
		}
	}

	if len(resultQueues) == 0 {
		if binding.AlternateExchange != nil {
			// Check if we've already visited this alternate exchange to prevent infinite recursion
			if !visitedExchanges[binding.AlternateExchange.ID] {
				// Mark this alternate exchange as visited
				visitedExchanges[binding.AlternateExchange.ID] = true

				queues, err := bo.GetQueuesFromExchange(ctx, binding.AlternateExchange.Code, routingKeyOrPatternOrQueueCode, message, binding.VNamespace, cf, cfs)
				if err != nil {
					return resultQueues, fmt.Errorf("failed to get queues from alternate exchange: %w", err)
				} else {
					resultQueues = append(resultQueues, queues...)
				}
			}
		}
	}
	return resultQueues, nil
}

// Check if binding matches routing criteria based on exchange type
func (bo *ExchangeBO) matchesRoutingCriteria(binding models.Binding, routingKeyOrPatternOrQueueCode string, message models.QueueMessage) bool {
	if binding.Exchange == nil {
		return false
	}

	switch binding.Exchange.Type {
	case models.Direct:
		// For direct exchanges, match exact routing key
		return binding.RoutingKey == routingKeyOrPatternOrQueueCode

	case models.Fanout:
		// Fanout exchanges route to all bound queues regardless of routing key
		return true

	case models.Topic:
		// For topic exchanges, match pattern
		return bo.matchesTopicPattern(binding.Pattern, routingKeyOrPatternOrQueueCode)

	case models.Headers:
		// For headers exchanges, match based on headers and XMatch type
		return bo.matchesHeaders(binding, message)

	default:
		return false
	}
}

// Match topic pattern (AMQP-compliant implementation)
func (bo *ExchangeBO) matchesTopicPattern(pattern, routingKey string) bool {
	// Handle special cases
	if pattern == "" {
		return routingKey == ""
	}
	if pattern == "#" {
		return true // # matches everything
	}

	// Split pattern and routing key by dots
	patternWords := strings.Split(pattern, ".")
	routingWords := strings.Split(routingKey, ".")

	return bo.matchTopicWords(patternWords, routingWords)
}

// Helper function to match topic words recursively
func (bo *ExchangeBO) matchTopicWords(patternWords, routingWords []string) bool {
	// Base cases
	if len(patternWords) == 0 && len(routingWords) == 0 {
		return true // Both empty, match
	}
	if len(patternWords) == 0 {
		return false // Pattern exhausted but routing key has more words
	}

	currentPattern := patternWords[0]
	remainingPattern := patternWords[1:]

	switch currentPattern {
	case "#":
		// # can match zero or more words
		if len(remainingPattern) == 0 {
			return true // # at the end matches everything remaining
		}

		// Try matching # with 0, 1, 2, ... words from routing key
		for i := 0; i <= len(routingWords); i++ {
			if bo.matchTopicWords(remainingPattern, routingWords[i:]) {
				return true
			}
		}
		return false

	case "*":
		// * must match exactly one word
		if len(routingWords) == 0 {
			return false // No word to match
		}
		// Match one word and continue with remaining
		return bo.matchTopicWords(remainingPattern, routingWords[1:])

	default:
		// Literal word - must match exactly
		if len(routingWords) == 0 || routingWords[0] != currentPattern {
			return false
		}
		// Exact match, continue with remaining
		return bo.matchTopicWords(remainingPattern, routingWords[1:])
	}
}

// Match headers based on XMatch type
func (bo *ExchangeBO) matchesHeaders(binding models.Binding, message models.QueueMessage) bool {
	if len(binding.Headers) == 0 {
		return true // No headers to match
	}

	messageHeaders := message.Headers

	switch binding.XMatch {
	case models.XMatchTypeAll:
		// All binding headers must match
		for key, value := range binding.Headers {
			if messageHeaders[key] != value {
				return false
			}
		}
		return true

	case models.XMatchTypeAny:
		// At least one binding header must match
		for key, value := range binding.Headers {
			if messageHeaders[key] == value {
				return true
			}
		}
		return false

	default:
		return false
	}
}

// Find queues by pattern for dynamic bindings
func (bo *ExchangeBO) findQueuesByPattern(ctx context.Context, pattern string, message models.QueueMessage, vnamespace, cf, cfs string, visitedExchanges map[string]bool) ([]models.Queue, error) {
	_ = ctx              // TODO: Use context for timeout
	_ = pattern          // TODO: Implement pattern matching
	_ = vnamespace       // TODO: Filter by namespace
	_ = cf               // TODO: Use CF parameter
	_ = cfs              // TODO: Use CFS parameter
	_ = visitedExchanges // To prevent infinite recursion in case of exchanges
	_ = message          // TODO: Use message for headers matching if needed

	// This would use a queue service to find queues matching the pattern
	// For now, return empty slice as placeholder
	return []models.Queue{}, nil
}

func (bo *ExchangeBO) deduplicateQueues(queues []models.Queue) []models.Queue {
	seen := make(map[string]bool)
	var result []models.Queue

	for _, queue := range queues {
		if !seen[queue.ID] {
			seen[queue.ID] = true
			result = append(result, queue)
		}
	}

	return result
}
