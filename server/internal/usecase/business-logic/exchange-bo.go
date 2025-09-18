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

	fmt.Println("Publishing message with ID:", message.MessageID, "to exchange:", exchangeCode, "with routingKeyOrPatternOrQueueCode:", routingKeyOrPatternOrQueueCode, "and vnamespace:", vnamespace)

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
		bo.Config.Logger.Error().Msg("Enqueue command failed")
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
	visitedExchanges[binding.ExchangeID] = true

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
	queues, err := bo.findQueuesByPattern(ctx, routingKeyOrPatternOrQueueCode, message, binding.VNamespace, cf, cfs, visitedExchanges)
	if err != nil {
		return resultQueues, fmt.Errorf("failed to find queues by pattern: %w", err)
	}

	resultQueues = append(resultQueues, queues...)

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
