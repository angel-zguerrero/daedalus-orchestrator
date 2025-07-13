package tenant_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(FindTenantCommand{})
}

// FindTenantCommand represents a command to authenticate a user.
type FindTenantCommand struct {
	TenantID   string
	TenantCode string
}

func (cmd *FindTenantCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	tenantInMasterRepo, err := db.NewTenantInMasterRepository(uow, idFactory) // Passing nil for IDGeneratorFactory
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	if cmd.TenantID != "" {
		tenantInMasterFound, err := tenantInMasterRepo.GetTenantInMasterByTenantId(cmd.TenantID, now)

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		commandResult.Result = tenantInMasterFound

		return *commandResult
	} else {
		tenantInMasterFound, err := tenantInMasterRepo.GetTenantInMasterByTenantCode(cmd.TenantCode, now)

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		commandResult.Result = tenantInMasterFound

		return *commandResult
	}

}
