package node_scheduler

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateNodeSchedulersCommand{})
	gob.Register(db.FindResult[models.NodeScheduler]{})

}

// PaginateNodeSchedulersCommand represents a command to authenticate a user.
type PaginateNodeSchedulersCommand struct {
	Cursor                  string
	PageSize                int
	Q                       string
	BalancingId             string
	ConnectionStatus        models.ConnectionStatus
	AssignedTenantNodeIndex int
}

func (cmd *PaginateNodeSchedulersCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	nodeSchedulerInMasterRepo, err := db.NewNodeSchedulerRepository(uow, idFactory) // Passing nil for IDGeneratorFactory
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	nodeSchedulerInMasterFound, err := nodeSchedulerInMasterRepo.Paginate(cmd.Q, cmd.BalancingId, cmd.ConnectionStatus, cmd.AssignedTenantNodeIndex, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	commandResult.Result = nodeSchedulerInMasterFound

	return *commandResult
}
