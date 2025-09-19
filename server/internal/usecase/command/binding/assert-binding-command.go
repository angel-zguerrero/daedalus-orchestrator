package binding

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"errors"
	"fmt"
	"time"
)

func init() {
	gob.Register(AssertBindingCommand{})
	gob.Register(models.Binding{})
	gob.Register(models.RoutingHeader{})
	gob.Register(map[string]string{})
}

type AssertBindingCommand struct {
	NewBindingID          string
	Code                  string
	QueueCode             string
	ExchangeCode          string
	TargetExchangeCode    string
	AlternateExchangeCode string
	VNamespace            string
	RoutingKey            string
	Pattern               string
	XMatch                models.XMatchType
	BindingType           models.BindingType
	TargetExchangeType    models.TargetExchangeType
	CF                    string
	CFS                   string
	Headers               map[string]string // Headers for routing, used only for Headers exchange type
}

func (cmd *AssertBindingCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	// Validate required fields
	if cmd.NewBindingID == "" {
		commandResult.Error = "NewBindingID is required"
		return *commandResult
	}

	if cmd.Code == "" {
		commandResult.Error = "Code is required"
		return *commandResult
	}

	if cmd.ExchangeCode == "" {
		commandResult.Error = "ExchangeCode is required"
		return *commandResult
	}

	// Validate TargetExchangeType and its dependencies
	if cmd.TargetExchangeType == "" {
		cmd.TargetExchangeType = models.TargetExchangeTypeQueue // Default to queue
	}

	// Validate TargetExchangeType specific requirements
	if cmd.TargetExchangeType == models.TargetExchangeTypeQueue {
		// For queue target, QueueCode is required for classic bindings
		if cmd.BindingType == models.BindingTypeClassic && cmd.QueueCode == "" {
			commandResult.Error = "QueueCode is required for classic bindings when TargetExchangeType is queue"
			return *commandResult
		}
		// TargetExchangeCode should not be specified when targeting a queue
		if cmd.TargetExchangeCode != "" {
			commandResult.Error = "TargetExchangeCode should not be specified when TargetExchangeType is queue"
			return *commandResult
		}
	} else if cmd.TargetExchangeType == models.TargetExchangeTypeExchange {
		// For exchange target, TargetExchangeCode is required
		if cmd.TargetExchangeCode == "" {
			commandResult.Error = "TargetExchangeCode is required when TargetExchangeType is exchange"
			return *commandResult
		}
		// QueueCode should not be specified when targeting an exchange
		if cmd.QueueCode != "" {
			commandResult.Error = "QueueCode should not be specified when TargetExchangeType is exchange"
			return *commandResult
		}
		// Exchange targets are only valid for classic bindings
		if cmd.BindingType == models.BindingTypeDynamic {
			commandResult.Error = "Exchange targets are not supported for dynamic bindings"
			return *commandResult
		}
	}

	// Validate legacy BindingType specific requirements (for backward compatibility)
	if cmd.BindingType == models.BindingTypeClassic {
		if cmd.TargetExchangeType == models.TargetExchangeTypeQueue && cmd.QueueCode == "" {
			commandResult.Error = "QueueCode is required for classic bindings when targeting a queue"
			return *commandResult
		}
	} else if cmd.BindingType == models.BindingTypeDynamic {
		// For dynamic bindings, QueueCode should be empty and only queue targets are allowed
		if cmd.QueueCode != "" {
			commandResult.Error = "QueueCode should not be specified for dynamic bindings"
			return *commandResult
		}
		if cmd.TargetExchangeType != models.TargetExchangeTypeQueue {
			commandResult.Error = "Dynamic bindings only support queue targets"
			return *commandResult
		}
	}

	// Set default VNamespace if not provided
	if cmd.VNamespace == "" {
		cmd.VNamespace = "default"
	}

	// Set default BindingType if not provided
	if cmd.BindingType == "" {
		cmd.BindingType = models.BindingTypeClassic
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	bindingRepo, err := db.NewBindingRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Get repositories for validation
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

	// Initialize routing headers repository for headers management
	routingHeadersRepo, err := db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
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
		commandResult.Error = "Exchange with Code '" + cmd.ExchangeCode + "' in VNamespace '" + cmd.VNamespace + "' does not exist"
		return *commandResult
	}

	// Validate that dynamic bindings cannot use Fanout or Topic exchanges
	if cmd.BindingType == models.BindingTypeDynamic && (exchange.Type == models.Fanout || exchange.Type == models.Topic) {
		commandResult.Error = fmt.Sprintf("Dynamic bindings cannot use %s exchanges as they don't support this routing type", exchange.Type)
		return *commandResult
	}

	// Find Queue by Code and VNamespace (only for queue targets in classic bindings)
	var queue *models.Queue
	if cmd.TargetExchangeType == models.TargetExchangeTypeQueue && cmd.BindingType == models.BindingTypeClassic {
		queue, err = queueRepo.GetQueueByCode(cmd.QueueCode, cmd.VNamespace, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		if queue == nil {
			commandResult.Error = "Queue with Code '" + cmd.QueueCode + "' in VNamespace '" + cmd.VNamespace + "' does not exist"
			return *commandResult
		}
	}

	// Find Target Exchange by Code and VNamespace (only for exchange targets)
	var targetExchange *models.Exchange
	if cmd.TargetExchangeType == models.TargetExchangeTypeExchange {
		targetExchange, err = exchangeRepo.GetExchangeByCode(cmd.TargetExchangeCode, cmd.VNamespace, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		if targetExchange == nil {
			commandResult.Error = "Target Exchange with Code '" + cmd.TargetExchangeCode + "' in VNamespace '" + cmd.VNamespace + "' does not exist"
			return *commandResult
		}
	}

	// Find Alternate Exchange by Code and VNamespace (optional)
	var alternateExchange *models.Exchange
	if cmd.AlternateExchangeCode != "" {
		alternateExchange, err = exchangeRepo.GetExchangeByCode(cmd.AlternateExchangeCode, cmd.VNamespace, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		if alternateExchange == nil {
			commandResult.Error = "Alternate Exchange with Code '" + cmd.AlternateExchangeCode + "' in VNamespace '" + cmd.VNamespace + "' does not exist"
			return *commandResult
		}

		// Validate that Alternate Exchange must be of type Fanout
		if alternateExchange.Type != models.Fanout {
			commandResult.Error = "Alternate Exchange must be of type 'fanout', got '" + string(alternateExchange.Type) + "'"
			return *commandResult
		}
	}

	// Validate binding parameters according to Exchange Type
	err = cmd.validateBindingParams(exchange.Type)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// For classic bindings with queue targets, check if there's already a binding between this exchange and queue
	if cmd.BindingType == models.BindingTypeClassic && cmd.TargetExchangeType == models.TargetExchangeTypeQueue && queue != nil {
		existingClassicBinding, err := bindingRepo.GetBindingByExchangeAndQueue(exchange.ID, queue.ID, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		if existingClassicBinding != nil && existingClassicBinding.Code != cmd.Code {
			commandResult.Error = "A classic binding between exchange '" + cmd.ExchangeCode + "' and queue '" + cmd.QueueCode + "' already exists with Code '" + existingClassicBinding.Code + "'"
			return *commandResult
		}
	}

	// For classic bindings with exchange targets, check if there's already a binding between this exchange and target exchange
	if cmd.BindingType == models.BindingTypeClassic && cmd.TargetExchangeType == models.TargetExchangeTypeExchange && targetExchange != nil {
		existingExchangeBinding, err := bindingRepo.GetBindingByExchangeAndTargetExchange(exchange.ID, targetExchange.ID, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		if existingExchangeBinding != nil && existingExchangeBinding.Code != cmd.Code {
			commandResult.Error = "A classic binding between exchange '" + cmd.ExchangeCode + "' and target exchange '" + cmd.TargetExchangeCode + "' already exists with Code '" + existingExchangeBinding.Code + "'"
			return *commandResult
		}
	}

	// Look for existing binding by Code and VNamespace
	existing, err := bindingRepo.GetBindingByCode(cmd.Code, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var binding models.Binding
	newBindingCreated := false

	if existing != nil {
		// Update existing binding
		binding = *existing
		binding.RoutingKey = cmd.RoutingKey
		binding.Pattern = cmd.Pattern
		binding.XMatch = cmd.XMatch
		binding.BindingType = cmd.BindingType
		binding.TargetExchangeType = cmd.TargetExchangeType

		// Update ExchangeID, QueueID, TargetExchangeID and AlternateExchangeID
		binding.ExchangeID = exchange.ID

		if cmd.TargetExchangeType == models.TargetExchangeTypeQueue {
			if queue != nil {
				binding.QueueID = queue.ID
			} else {
				binding.QueueID = "" // For dynamic bindings
			}
			binding.TargetExchangeID = "" // Clear target exchange when targeting queue
		} else {
			binding.QueueID = "" // Clear queue when targeting exchange
			if targetExchange != nil {
				binding.TargetExchangeID = targetExchange.ID
			}
		}

		// Set alternate exchange (optional)
		if alternateExchange != nil {
			binding.AlternateExchangeID = alternateExchange.ID
		} else {
			binding.AlternateExchangeID = ""
		}

		_, err = bindingRepo.UpdateBinding(&binding, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	} else {
		// Create new binding
		queueID := ""
		targetExchangeID := ""
		alternateExchangeID := ""

		if cmd.TargetExchangeType == models.TargetExchangeTypeQueue {
			if queue != nil {
				queueID = queue.ID
			}
		} else {
			if targetExchange != nil {
				targetExchangeID = targetExchange.ID
			}
		}

		if alternateExchange != nil {
			alternateExchangeID = alternateExchange.ID
		}

		binding = models.Binding{
			ID:                  cmd.NewBindingID,
			Code:                cmd.Code,
			VNamespace:          cmd.VNamespace,
			ExchangeID:          exchange.ID,
			QueueID:             queueID,
			TargetExchangeID:    targetExchangeID,
			AlternateExchangeID: alternateExchangeID,
			TargetExchangeType:  cmd.TargetExchangeType,
			RoutingKey:          cmd.RoutingKey,
			Pattern:             cmd.Pattern,
			XMatch:              cmd.XMatch,
			BindingType:         cmd.BindingType,
			CreatedAt:           now,
			UpdatedAt:           now,
		}

		// Set default XMatch if not specified for headers exchanges
		if binding.XMatch == "" {
			binding.XMatch = models.XMatchTypeAll
		}

		// Upsert VNamespace if it doesn't exist
		existingVNamespace, err := vNamespaceRepo.GetVNamespaceByName(cmd.VNamespace, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		if existingVNamespace == nil {
			// Create new VNamespace
			vNamespace := models.VNamespace{
				ID:   idFactory.GenerateID(),
				Name: cmd.VNamespace,
			}
			_, err = vNamespaceRepo.CreateVNamespace(&vNamespace, now)
			if err != nil {
				commandResult.Error = err.Error()
				return *commandResult
			}
		}

		_, err = bindingRepo.CreateBinding(&binding, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		newBindingCreated = true
	}

	// Update tenant summary if a new binding was created
	if newBindingCreated {
		err = tenantSummaryRepo.UpdateCounters(cmd.CFS, 0, 0, 0, 1, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	// Handle routing headers for Headers exchange type in classic bindings only
	if exchange.Type == models.Headers && cmd.BindingType == models.BindingTypeClassic && cmd.Headers != nil && len(cmd.Headers) > 0 {
		err = cmd.upsertRoutingHeaders(routingHeadersRepo, binding.ID, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	commandResult.Result = binding
	return *commandResult
}

// validateBindingParams validates binding parameters according to Exchange Type and BindingType
func (cmd *AssertBindingCommand) validateBindingParams(exchangeType models.ExchangeType) error {
	switch exchangeType {
	case models.Direct:
		// Direct exchanges require RoutingKey only for classic bindings
		// For dynamic bindings, RoutingKey is ignored as queue is found by code
		if cmd.BindingType == models.BindingTypeClassic && cmd.RoutingKey == "" {
			return errors.New("RoutingKey is required for Direct exchanges in classic bindings")
		}
		// Pattern and XMatch should be empty for direct exchanges
		if cmd.Pattern != "" {
			return errors.New("Pattern should not be specified for Direct exchanges")
		}
		if cmd.XMatch != "" && cmd.XMatch != models.XMatchTypeAll {
			return errors.New("XMatch should not be specified for Direct exchanges")
		}

	case models.Topic:
		// Topic exchanges require Pattern for all binding types
		// For classic bindings, Pattern is used for message routing
		// For dynamic bindings, Pattern is used for automatic queue discovery
		if cmd.Pattern == "" {
			return errors.New("Pattern is required for Topic exchanges")
		}
		// RoutingKey and XMatch should be empty for topic exchanges
		if cmd.RoutingKey != "" {
			return errors.New("RoutingKey should not be specified for Topic exchanges")
		}
		if cmd.XMatch != "" && cmd.XMatch != models.XMatchTypeAll {
			return errors.New("XMatch should not be specified for Topic exchanges")
		}

	case models.Headers:
		// Headers exchanges require XMatch
		if cmd.XMatch == "" {
			cmd.XMatch = models.XMatchTypeAll // Set default
		}
		// For dynamic bindings, queues are determined automatically based on message headers
		// and queue header conditions, so only XMatch is needed
		// For classic bindings, routing headers are required
		if cmd.BindingType == models.BindingTypeClassic {
			// Headers are validated separately in the main command handler
		}
		// RoutingKey and Pattern should be empty for headers exchanges
		if cmd.RoutingKey != "" {
			return errors.New("RoutingKey should not be specified for Headers exchanges")
		}
		if cmd.Pattern != "" {
			return errors.New("Pattern should not be specified for Headers exchanges")
		}

	case models.Fanout:
		// Fanout exchanges don't require any specific parameters
		// All fields can be empty as fanout broadcasts to all bound queues
		// But warn if any are provided as they won't be used
		if cmd.RoutingKey != "" {
			return errors.New("RoutingKey is not used for Fanout exchanges")
		}
		if cmd.Pattern != "" {
			return errors.New("Pattern is not used for Fanout exchanges")
		}
		if cmd.XMatch != "" && cmd.XMatch != models.XMatchTypeAll {
			return errors.New("XMatch is not used for Fanout exchanges")
		}

	default:
		return errors.New("unsupported Exchange Type: " + string(exchangeType))
	}

	return nil
}

// upsertRoutingHeaders creates or updates routing headers for a binding
func (cmd *AssertBindingCommand) upsertRoutingHeaders(routingHeadersRepo *db.RoutingHeadersRepository, bindingID string, now time.Time) error {
	// Get existing headers for this binding
	existingHeaders, err := routingHeadersRepo.GetRoutingHeadersByBinding(bindingID, now)
	if err != nil {
		return err
	}

	// Create a map of existing headers by key for fast lookup
	existingByKey := make(map[string]*models.RoutingHeader)
	if existingHeaders != nil {
		for i := range existingHeaders.Entities {
			header := &existingHeaders.Entities[i]
			existingByKey[header.Key] = header
		}
	}

	// Track which keys we're processing to identify headers to delete
	processedKeys := make(map[string]bool)

	// Create or update headers from the map
	for key, value := range cmd.Headers {
		// Generate unique ID by combining binding ID and header key
		headerID := bindingID + "_" + key
		processedKeys[key] = true

		if existingHeader, exists := existingByKey[key]; exists {
			// Update existing header
			existingHeader.Value = value
			existingHeader.UpdatedAt = now

			_, err := routingHeadersRepo.UpdateRoutingHeader(existingHeader, now)
			if err != nil {
				return err
			}
		} else {
			// Create new header
			routingHeader := &models.RoutingHeader{
				ID:         headerID,
				VNamespace: cmd.VNamespace,
				BindingID:  bindingID,
				Key:        key,
				Value:      value,
				CreatedAt:  now,
				UpdatedAt:  now,
				HeaderType: models.HeaderTypeBinding,
			}

			_, err := routingHeadersRepo.CreateRoutingHeader(routingHeader, now)
			if err != nil {
				return err
			}
		}
	}

	// Delete headers that are no longer present in the new map
	for key, header := range existingByKey {
		if !processedKeys[key] {
			_, err := routingHeadersRepo.DeleteRoutingHeader(header.ID, now)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
