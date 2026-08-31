package binding

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(DeleteBindingCommand{})
}

type DeleteBindingCommand struct {
	Code       string
	VNamespace string
	CF         string
	CFS        string
}

func (cmd *DeleteBindingCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DefaultIDGeneratorFactory{}

	// Get repositories
	bindingRepo, err := db.NewBindingRepository(uow, idFactory, cmd.CF, cmd.CFS)
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

	// Find binding by Code and VNamespace
	binding, err := bindingRepo.GetBindingByCode(cmd.Code, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	if binding == nil {
		commandResult.Error = "binding not found"
		return *commandResult
	}

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

	// Delete the binding by ID
	deleted, err := bindingRepo.DeleteBinding(binding.ID, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if !deleted {
		commandResult.Error = "binding not found or could not be deleted"
		return *commandResult
	}

	// Update tenant summary
	err = tenantSummaryRepo.UpdateCounters(cmd.CFS, 0, 0, 0, -1, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// ── Maintain the Route Table ─────────────────────────────────────────────
	if err := cmd.removeFromRouteTable(uow, binding, now); err != nil {
		commandResult.Error = "error updating route table: " + err.Error()
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}

// removeFromRouteTable cleans up the route table when a binding is deleted.
func (cmd *DeleteBindingCommand) removeFromRouteTable(uow *db.UnitOfWork, binding *models.Binding, now time.Time) error {
	// We need the exchange to know its type
	idFactory := &db.DefaultIDGeneratorFactory{}
	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		return err
	}
	
	exchange, err := exchangeRepo.GetExchangeById(binding.ExchangeID, now)
	if err != nil || exchange == nil {
		// If exchange is gone or error, we might leave orphans, but typically exchange deletion 
		// cascades or we ignore. For safety, proceed only if exchange exists.
		return nil
	}

	rt := db.NewRouteTable(uow.KVStore, cmd.CF, cmd.CFS, "admin_schema")
	batch := db.NewWriteBatch()

	var target string
	if binding.TargetExchangeType == models.TargetExchangeTypeExchange && binding.TargetExchangeID != "" {
		target = "e:" + binding.TargetExchangeID
	} else if binding.QueueID != "" {
		target = "q:" + binding.QueueID
	}

	if target == "" {
		return nil
	}

	switch exchange.Type {
	case models.Direct:
		if binding.BindingType == models.BindingTypeClassic && binding.RoutingKey != "" {
			if err := rt.RemoveDirectRoute(batch, exchange.ID, binding.RoutingKey, target, now); err != nil {
				return err
			}
		} else if binding.BindingType == models.BindingTypeDynamic {
			if err := rt.RemoveDynamicRoute(batch, exchange.ID, now); err != nil {
				return err
			}
		}

	case models.Fanout:
		if binding.BindingType == models.BindingTypeClassic {
			if err := rt.RemoveFanoutRoute(batch, exchange.ID, target, now); err != nil {
				return err
			}
		} else if binding.BindingType == models.BindingTypeDynamic {
			if err := rt.RemoveDynamicRoute(batch, exchange.ID, now); err != nil {
				return err
			}
		}

	case models.Topic:
		if binding.BindingType == models.BindingTypeClassic {
			if err := rt.RemoveTopicPattern(batch, exchange.ID, binding.ID, now); err != nil {
				return err
			}
		} else if binding.BindingType == models.BindingTypeDynamic {
			if err := rt.RemoveDynamicRoute(batch, exchange.ID, now); err != nil {
				return err
			}
		}

	case models.Headers:
		if binding.BindingType == models.BindingTypeClassic {
			if err := rt.RemoveHeadersBinding(batch, exchange.ID, binding.ID, now); err != nil {
				return err
			}
		} else if binding.BindingType == models.BindingTypeDynamic {
			if err := rt.RemoveDynamicRoute(batch, exchange.ID, now); err != nil {
				return err
			}
		}
	}

	if batch.Count() > 0 {
		return uow.KVStore.Write(batch)
	}
	return nil
}



