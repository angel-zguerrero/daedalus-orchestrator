package node_scheduler

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateNodeSchedulersAssignedTenantNodeIndexCommand{})
	gob.Register(db.FindResult[models.NodeScheduler]{})

}

// PaginateNodeSchedulersAssignedTenantNodeIndexCommand represents a command to authenticate a user.
type PaginateNodeSchedulersAssignedTenantNodeIndexCommand struct {
	Cursor                  string
	PageSize                int
	Q                       string
	AssignedTenantNodeIndex int
}

func (cmd *PaginateNodeSchedulersAssignedTenantNodeIndexCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	nodeSchedulerInMasterRepo, err := db.NewNodeSchedulerRepository(uow, idFactory) // Passing nil for IDGeneratorFactory
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	nodeSchedulerInMasterFound, err := nodeSchedulerInMasterRepo.PaginateUsingAssignedTenantNodeIndex(cmd.Q, cmd.AssignedTenantNodeIndex, cmd.PageSize, cmd.Cursor, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}
	commandResult.Result = nodeSchedulerInMasterFound

	return *commandResult
}
