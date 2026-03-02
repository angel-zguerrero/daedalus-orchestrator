package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	commands "deadalus-orch/server/internal/usecase/command"
	binding_command "deadalus-orch/server/internal/usecase/command/binding"
	exchange_command "deadalus-orch/server/internal/usecase/command/exchange"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	header_command "deadalus-orch/server/internal/usecase/command/header"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	tenant_summary_command "deadalus-orch/server/internal/usecase/command/tenant-summary"
	vnamespace_command "deadalus-orch/server/internal/usecase/command/vnamespace"
	"time"

	"github.com/lni/dragonboat/v4/statemachine"
)

type TenantKVBaseStateMachine struct {
}

// OpenTenantDBFunc is a function variable that points to db.OpenTenantDB by default.
// It allows for replacing the actual tenant database opening logic with a mock
// implementation during testing. This is a common pattern for dependency injection.
var OpenTenantDBFunc = db.OpenTenantDB

func (r *TenantKVBaseStateMachine) OpenDB(dbPath string) (db.KVStore, error) {
	return OpenTenantDBFunc(dbPath)
}

func (r *TenantKVBaseStateMachine) Lookup(cmd any, uow *db.UnitOfWork, now time.Time) commands.CommandResult {

	findExchangeCommand, ok := cmd.(exchange_command.FindExchangeCommand)
	if ok {
		return findExchangeCommand.Execute(uow, now)
	}

	paginateExchangesCommand, ok := cmd.(exchange_command.PaginateExchangesCommand)
	if ok {
		return paginateExchangesCommand.Execute(uow, now)
	}

	findQueueCommand, ok := cmd.(queue_command.FindQueueCommand)
	if ok {
		return findQueueCommand.Execute(uow, now)
	}

	paginateQueuesCommand, ok := cmd.(queue_command.PaginateQueuesCommand)
	if ok {
		return paginateQueuesCommand.Execute(uow, now)
	}

	paginateQueuesWithFilterCommand, ok := cmd.(queue_command.PaginateQueuesWithFilterCommand)
	if ok {
		return paginateQueuesWithFilterCommand.Execute(uow, now)
	}

	findBindingCommand, ok := cmd.(binding_command.FindBindingCommand)
	if ok {
		return findBindingCommand.Execute(uow, now)
	}

	paginateBindingsCommand, ok := cmd.(binding_command.PaginateBindingsCommand)
	if ok {
		return paginateBindingsCommand.Execute(uow, now)
	}

	paginateByExchangeBindingsCommand, ok := cmd.(binding_command.PaginateByExchangeBindingsCommand)
	if ok {
		return paginateByExchangeBindingsCommand.Execute(uow, now)
	}

	paginateVNamespacesCommand, ok := cmd.(vnamespace_command.PaginateVNamespacesCommand)
	if ok {
		return paginateVNamespacesCommand.Execute(uow, now)
	}

	paginateVNamespacesWithFilterCommand, ok := cmd.(vnamespace_command.PaginateVNamespacesWithFilterCommand)
	if ok {
		return paginateVNamespacesWithFilterCommand.Execute(uow, now)
	}

	paginateTenantUpdatedAtFromCommand, ok := cmd.(tenant_summary_command.PaginateTenantUpdatedAtFromCommand)
	if ok {
		return paginateTenantUpdatedAtFromCommand.Execute(uow, now)
	}

	getLastUpdateAtFromCommand, ok := cmd.(tenant_summary_command.GetLastUpdateAtFromCommand)
	if ok {
		return getLastUpdateAtFromCommand.Execute(uow, now)
	}

	getTenantSummaryCommand, ok := cmd.(tenant_summary_command.GetTenantSummaryCommand)
	if ok {
		return getTenantSummaryCommand.Execute(uow, now)
	}

	findQueueByIDCommand, ok := cmd.(queue_command.FindQueueByIDCommand)
	if ok {
		return findQueueByIDCommand.Execute(uow, now)
	}

	findQueueByIDsCommand, ok := cmd.(queue_command.FindQueueByIDsCommand)
	if ok {
		return findQueueByIDsCommand.Execute(uow, now)
	}

	findExchangeByIDCommand, ok := cmd.(exchange_command.FindExchangeByIDCommand)
	if ok {
		return findExchangeByIDCommand.Execute(uow, now)
	}

	listHeadersCommand, ok := cmd.(header_command.ListHeadersCommand)
	if ok {
		return listHeadersCommand.Execute(uow, now)
	}

	commandResult := &commands.CommandResult{}
	commandResult.Error = "invalid command type"
	return *commandResult
}

func (r *TenantKVBaseStateMachine) Update(cmd any, uow *db.UnitOfWork, now time.Time) commands.CommandResult {

	createColumnFamilyCommand, ok := cmd.(general_command.CreateColumnFamilyCommand)
	if ok {
		return createColumnFamilyCommand.Execute(uow, now)
	}

	deleteColumnFamilyCommand, ok := cmd.(general_command.DeleteColumnFamilyCommand)
	if ok {
		return deleteColumnFamilyCommand.Execute(uow, now)
	}

	deleteColumnFamilySectorCommand, ok := cmd.(general_command.DeleteColumnFamilySectorCommand)
	if ok {
		return deleteColumnFamilySectorCommand.Execute(uow, now)
	}

	AssertExchangeCommand, ok := cmd.(exchange_command.AssertExchangeCommand)
	if ok {
		return AssertExchangeCommand.Execute(uow, now)
	}

	deleteExchangeCommand, ok := cmd.(exchange_command.DeleteExchangeCommand)
	if ok {
		return deleteExchangeCommand.Execute(uow, now)
	}

	AssertQueueCommand, ok := cmd.(queue_command.AssertQueueCommand)
	if ok {
		return AssertQueueCommand.Execute(uow, now)
	}

	deleteQueueCommand, ok := cmd.(queue_command.DeleteQueueCommand)
	if ok {
		return deleteQueueCommand.Execute(uow, now)
	}

	assertBindingCommand, ok := cmd.(binding_command.AssertBindingCommand)
	if ok {
		return assertBindingCommand.Execute(uow, now)
	}

	deleteBindingCommand, ok := cmd.(binding_command.DeleteBindingCommand)
	if ok {
		return deleteBindingCommand.Execute(uow, now)
	}

	refreshLastUpdateAtFromCommand, ok := cmd.(tenant_summary_command.RefreshLastUpdateAtFromCommand)
	if ok {
		return refreshLastUpdateAtFromCommand.Execute(uow, now)
	}

	updateTenantSummaryCommand, ok := cmd.(tenant_summary_command.UpdateTenantSummaryCommand)
	if ok {
		return updateTenantSummaryCommand.Execute(uow, now)
	}

	enqueueCommand, ok := cmd.(queue_command.EnqueueCommand)
	if ok {
		return enqueueCommand.Execute(uow, now)
	}

	markQueuesAsDrainCommand, ok := cmd.(queue_command.MarkQueuesAsDrainCommand)
	if ok {
		return markQueuesAsDrainCommand.Execute(uow, now)
	}

	assignNodeSchedulerToQueuesCommand, ok := cmd.(queue_command.AssignNodeSchedulerToQueuesCommand)
	if ok {
		return assignNodeSchedulerToQueuesCommand.Execute(uow, now)
	}

	commandResult := &commands.CommandResult{}
	commandResult.Error = "invalid command type"
	return *commandResult
}

func NewTenantKVStateMachine(pathProvider db.PathProvider) func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	return func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
		return NewKVStateMachine(clusterID, nodeID, &TenantKVBaseStateMachine{}, KVBaseStateMachineConfig{
			TTLInternalError: config.GlobalConfiguration.TTLInternalError,
			PathProvider:     pathProvider,
		})
	}

}
