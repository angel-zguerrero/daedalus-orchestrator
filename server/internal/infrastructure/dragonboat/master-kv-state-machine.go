package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	commands "deadalus-orch/server/internal/usecase/command"
	auth_command "deadalus-orch/server/internal/usecase/command/auth"
	job_worker_command "deadalus-orch/server/internal/usecase/command/job-worker"

	tenant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"time"

	"github.com/lni/dragonboat/v4/statemachine"
)

type MasterKVDBStateMachine struct {
}

func (r *MasterKVDBStateMachine) OpenDB(sharedProvider *db.SharedDBProvider, pathProvider db.PathProvider) (db.KVStore, error) {
	return sharedProvider.Acquire(pathProvider)
}

// BelongsToShard returns true if the given column family belongs to the master shard.
// The master shard owns the "admin" CF and event CFs (master-events).
// Dynamic tenant CFs (cf-n-X) do NOT belong to the master shard.
func (r *MasterKVDBStateMachine) BelongsToShard(cfName string) bool {
	switch cfName {
	case db.AdminFC, db.MasterEventFC, db.DefaultFC:
		return true
	default:
		return false
	}
}

func (r *MasterKVDBStateMachine) Lookup(input any, uow *db.UnitOfWork, now time.Time) commands.CommandResult {

	loginCmd, ok := input.(auth_command.LoginCommand)
	if ok {
		return loginCmd.Execute(uow, now)
	}

	checkRootUserExistsCommand, ok := input.(auth_command.CheckRootUserExistsCommand)
	if ok {
		return checkRootUserExistsCommand.Execute(uow, now)
	}

	checkSessionExistsCommand, ok := input.(auth_command.CheckSessionExistsCommand)
	if ok {
		return checkSessionExistsCommand.Execute(uow, now)
	}
	paginateTenantsCommand, ok := input.(tenant_command.PaginateTenantsCommand)
	if ok {
		return paginateTenantsCommand.Execute(uow, now)
	}

	paginateTenantsWithFilterCommand, ok := input.(tenant_command.PaginateTenantsWithFilterCommand)
	if ok {
		return paginateTenantsWithFilterCommand.Execute(uow, now)
	}

	findTenantCommand, ok := input.(tenant_command.FindTenantCommand)
	if ok {
		return findTenantCommand.Execute(uow, now)
	}



	paginateJobWorkersCommand, ok := input.(job_worker_command.PaginateJobWorkersCommand)
	if ok {
		return paginateJobWorkersCommand.Execute(uow, now)
	}

	getTenantSummaryCommand, ok := input.(tenant_command.GetTenantSummaryCommand)
	if ok {
		return getTenantSummaryCommand.Execute(uow, now)
	}

	getDashboardSummaryCommand, ok := input.(tenant_command.GetDashboardSummaryCommand)
	if ok {
		return getDashboardSummaryCommand.Execute(uow, now)
	}

	commandResult := &commands.CommandResult{}
	commandResult.Error = "invalid command type"

	return *commandResult
}

func (r *MasterKVDBStateMachine) Update(cmd any, uow *db.UnitOfWork, now time.Time) commands.CommandResult {
	bootstrapRootUserCmd, ok := cmd.(auth_command.BootstrapRootUserCommand)
	if ok {
		return bootstrapRootUserCmd.Execute(uow, now)
	}

	setupRootUserCmd, ok := cmd.(auth_command.SetupRootUserCommand)
	if ok {
		return setupRootUserCmd.Execute(uow, now)
	}

	registerSessionCommand, ok := cmd.(auth_command.RegisterSessionCommand)
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

	updateTenantSummaryCommand, ok := cmd.(tenant_command.UpdateTenantSummaryCommand)
	if ok {
		return updateTenantSummaryCommand.Execute(uow, now)
	}

	removeSessionCommand, ok := cmd.(auth_command.RemoveSessionCommand)
	if ok {
		return removeSessionCommand.Execute(uow, now)
	}



	upsertJobWorkerCommand, ok := cmd.(job_worker_command.UpsertJobWorkerCommand)
	if ok {
		return upsertJobWorkerCommand.Execute(uow, now)
	}

	updateDashboardSummaryCommand, ok := cmd.(tenant_command.UpdateDashboardSummaryCommand)
	if ok {
		return updateDashboardSummaryCommand.Execute(uow, now)
	}

	commandResult := &commands.CommandResult{}
	commandResult.Error = "invalid command type"
	return *commandResult
}

func NewMasterKVStateMachine(pathProvider db.PathProvider, sharedDBProvider *db.SharedDBProvider) func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	return func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
		return NewKVStateMachine(clusterID, nodeID, &MasterKVDBStateMachine{}, KVBaseStateMachineConfig{
			TTLInternalError: config.GlobalConfiguration.TTLInternalError,
			PathProvider:     pathProvider,
			SharedDBProvider: sharedDBProvider,
		})
	}

}
