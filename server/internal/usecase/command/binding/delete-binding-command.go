package binding

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
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
	err = tenantSummaryRepo.DecreaseBindingCount(cmd.CFS, 1, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = true
	return *commandResult
}
