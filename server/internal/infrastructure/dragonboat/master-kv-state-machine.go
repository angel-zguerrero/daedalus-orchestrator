package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	commands "deadalus-orch/server/internal/usecase/command"
	tenant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"time"

	"github.com/lni/dragonboat/v4/statemachine"
)

type MasterKVDBStateMachine struct {
}

func (r *MasterKVDBStateMachine) OpenDB(dbPath string) (db.KVStore, error) {
	return db.OpenMasterDB(dbPath)
}

func (r *MasterKVDBStateMachine) Lookup(input any, uow *db.UnitOfWork, now time.Time) commands.CommandResult {

	loginCmd, ok := input.(commands.LoginCommand)
	if ok {
		return loginCmd.Execute(uow, now)
	}

	checkSessionExistsCommand, ok := input.(commands.CheckSessionExistsCommand)
	if ok {
		return checkSessionExistsCommand.Execute(uow, now)
	}
	paginateTenantsCommand, ok := input.(tenant_command.PaginateTenantsCommand)
	if ok {
		return paginateTenantsCommand.Execute(uow, now)
	}

	findTenantCommand, ok := input.(tenant_command.FindTenantCommand)
	if ok {
		return findTenantCommand.Execute(uow, now)
	}

	commandResult := &commands.CommandResult{}
	commandResult.Error = "invalid command type"

	return *commandResult
}

func (r *MasterKVDBStateMachine) Update(cmd any, uow *db.UnitOfWork, now time.Time) commands.CommandResult {
	bootstrapRootUserCmd, ok := cmd.(commands.BootstrapRootUserCommand)
	if ok {
		return bootstrapRootUserCmd.Execute(uow, now)
	}

	registerSessionCommand, ok := cmd.(commands.RegisterSessionCommand)
	if ok {
		return registerSessionCommand.Execute(uow, now)
	}

	createTenantInMasterCommand, ok := cmd.(tenant_command.CreateTenantInMasterCommand)
	if ok {
		return createTenantInMasterCommand.Execute(uow, now)
	}

	assignToShardTenantInMasterCommand, ok := cmd.(tenant_command.AssignToShardTenantInMasterCommand)
	if ok {
		return assignToShardTenantInMasterCommand.Execute(uow, now)
	}

	markToDeletionTenantInMasterCommand, ok := cmd.(tenant_command.MarkToDeletionTenantInMasterCommand)
	if ok {
		return markToDeletionTenantInMasterCommand.Execute(uow, now)
	}

	deleteTenantInMasterCommand, ok := cmd.(tenant_command.DeleteTenantInMasterCommand)
	if ok {
		return deleteTenantInMasterCommand.Execute(uow, now)
	}

	removeSessionCommand, ok := cmd.(commands.RemoveSessionCommand)
	if ok {
		return removeSessionCommand.Execute(uow, now)
	}

	commandResult := &commands.CommandResult{}
	commandResult.Error = "invalid command type"
	return *commandResult
}

func NewMasterKVStateMachine(pathProvider db.PathProvider) func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	return func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
		return NewKVStateMachine(clusterID, nodeID, &MasterKVDBStateMachine{}, KVBaseStateMachineConfig{
			TTLInternalError: config.GlobalConfiguration.TTLInternalError,
			PathProvider:     pathProvider,
		})
	}

}
