package binding

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"errors"
	"time"
)

func init() {
	gob.Register(AssertBindingCommand{})
	gob.Register(models.Binding{})
}

type AssertBindingCommand struct {
	QueueCode    string
	ExchangeCode string
	VNamespace   string
	RoutingKey   string
	Pattern      string
	XMatch       models.XMatchType
	BindingType  models.BindingType
	CF           string
	CFS          string
}

func (cmd *AssertBindingCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	// Validate required fields
	if cmd.ExchangeCode == "" {
		commandResult.Error = "ExchangeCode is required"
		return *commandResult
	}

	if cmd.QueueCode == "" {
		commandResult.Error = "QueueCode is required"
		return *commandResult
	}

	// Set default VNamespace if not provided
	if cmd.VNamespace == "" {
		cmd.VNamespace = "default"
	}

	// Set default BindingType if not provided
	if cmd.BindingType == "" {
		cmd.BindingType = models.BindingTypeClassic
	}

	idFactory := &db.DefaultIDGeneratorFactory{}
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

	// Find Queue by Code and VNamespace
	queue, err := queueRepo.GetQueueByCode(cmd.QueueCode, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	if queue == nil {
		commandResult.Error = "Queue with Code '" + cmd.QueueCode + "' in VNamespace '" + cmd.VNamespace + "' does not exist"
		return *commandResult
	}

	// Validate binding parameters according to Exchange Type
	err = cmd.validateBindingParams(exchange.Type)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Look for existing binding by ExchangeID and QueueID
	existing, err := bindingRepo.GetBindingByExchangeAndQueue(exchange.ID, queue.ID, now)
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

		_, err = bindingRepo.UpdateBinding(&binding, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	} else {
		// Create new binding
		binding = models.Binding{
			ID:          idFactory.GenerateID(),
			VNamespace:  cmd.VNamespace,
			ExchangeID:  exchange.ID,
			QueueID:     queue.ID,
			RoutingKey:  cmd.RoutingKey,
			Pattern:     cmd.Pattern,
			XMatch:      cmd.XMatch,
			BindingType: cmd.BindingType,
			CreatedAt:   now,
			UpdatedAt:   now,
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
		err = tenantSummaryRepo.IncreaseBindingCount(cmd.CFS, 1, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
	}

	commandResult.Result = binding
	return *commandResult
}

// validateBindingParams validates binding parameters according to Exchange Type
func (cmd *AssertBindingCommand) validateBindingParams(exchangeType models.ExchangeType) error {
	switch exchangeType {
	case models.Direct:
		// Direct exchanges require RoutingKey
		if cmd.RoutingKey == "" {
			return errors.New("RoutingKey is required for Direct exchanges")
		}
		// Pattern and XMatch should be empty for direct exchanges
		if cmd.Pattern != "" {
			return errors.New("Pattern should not be specified for Direct exchanges")
		}
		if cmd.XMatch != "" && cmd.XMatch != models.XMatchTypeAll {
			return errors.New("XMatch should not be specified for Direct exchanges")
		}

	case models.Topic:
		// Topic exchanges require Pattern
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
