package general_command

import (
	commands "deadalus-orch/server/internal/usecase/command"
	auth_command "deadalus-orch/server/internal/usecase/command/auth"
	binding_command "deadalus-orch/server/internal/usecase/command/binding"
	exchange_command "deadalus-orch/server/internal/usecase/command/exchange"
	header_command "deadalus-orch/server/internal/usecase/command/header"
	jobworker_command "deadalus-orch/server/internal/usecase/command/job-worker"
	metrics_command "deadalus-orch/server/internal/usecase/command/metrics"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	tenant_summary_command "deadalus-orch/server/internal/usecase/command/tenant-summary"
	tentant_command "deadalus-orch/server/internal/usecase/command/tentant"
	user_command "deadalus-orch/server/internal/usecase/command/user"
	vnamespace_command "deadalus-orch/server/internal/usecase/command/vnamespace"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

var (
	repoRegistryMutex sync.RWMutex
	repoRegistry      = make(map[string]func() commands.Command)
)

// RegisterRepoCommand registers a factory function for a command type by its struct name.
func RegisterRepoCommand(name string, factory func() commands.Command) {
	repoRegistryMutex.Lock()
	defer repoRegistryMutex.Unlock()
	repoRegistry[name] = factory
}

// GetCommandTypeName returns the struct type name of the given command.
func GetCommandTypeName(cmd any) string {
	if cmd == nil {
		return ""
	}
	t := reflect.TypeOf(cmd)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// EncodeRepoCommand encodes any repository command into typeName and jsonBytes.
func EncodeRepoCommand(cmd any) (string, []byte, error) {
	if cmd == nil {
		return "", nil, fmt.Errorf("nil repository command")
	}
	// Extract inner command if wrapped in Repository_Command
	if rCmd, ok := cmd.(Repository_Command); ok {
		cmd = rCmd.CMD
	} else if rCmdPtr, ok := cmd.(*Repository_Command); ok && rCmdPtr != nil {
		cmd = rCmdPtr.CMD
	}

	typeName := GetCommandTypeName(cmd)
	if typeName == "" {
		return "", nil, fmt.Errorf("empty type name for repository command")
	}

	jsonBytes, err := json.Marshal(cmd)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal repo command %s: %w", typeName, err)
	}

	return typeName, jsonBytes, nil
}

// DecodeRepoCommand instantiates and unmarshals a repository command by typeName.
func DecodeRepoCommand(typeName string, jsonBytes []byte) (commands.Command, error) {
	repoRegistryMutex.RLock()
	factory, ok := repoRegistry[typeName]
	repoRegistryMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unregistered repository command type: %s", typeName)
	}

	cmdInst := factory()
	if err := json.Unmarshal(jsonBytes, cmdInst); err != nil {
		return nil, fmt.Errorf("failed to unmarshal repo command %s: %w", typeName, err)
	}

	return cmdInst, nil
}

func init() {
	// Auth commands
	RegisterRepoCommand("LoginCommand", func() commands.Command { return &auth_command.LoginCommand{} })
	RegisterRepoCommand("BootstrapRootUserCommand", func() commands.Command { return &auth_command.BootstrapRootUserCommand{} })
	RegisterRepoCommand("CheckDemoResolvedCommand", func() commands.Command { return &auth_command.CheckDemoResolvedCommand{} })
	RegisterRepoCommand("CheckRootUserExistsCommand", func() commands.Command { return &auth_command.CheckRootUserExistsCommand{} })
	RegisterRepoCommand("CheckSessionExistsCommand", func() commands.Command { return &auth_command.CheckSessionExistsCommand{} })
	RegisterRepoCommand("RegisterSessionCommand", func() commands.Command { return &auth_command.RegisterSessionCommand{} })
	RegisterRepoCommand("RemoveSessionCommand", func() commands.Command { return &auth_command.RemoveSessionCommand{} })
	RegisterRepoCommand("SetupRootUserCommand", func() commands.Command { return &auth_command.SetupRootUserCommand{} })

	// Tenant Summary commands
	RegisterRepoCommand("RefreshLastUpdateAtFromCommand", func() commands.Command { return &tenant_summary_command.RefreshLastUpdateAtFromCommand{} })
	RegisterRepoCommand("PaginateTenantUpdatedAtFromCommand", func() commands.Command { return &tenant_summary_command.PaginateTenantUpdatedAtFromCommand{} })
	RegisterRepoCommand("GetTenantSummaryCommand", func() commands.Command { return &tenant_summary_command.GetTenantSummaryCommand{} })
	RegisterRepoCommand("UpdateTenantSummaryCommand", func() commands.Command { return &tenant_summary_command.UpdateTenantSummaryCommand{} })

	// Tenant commands
	RegisterRepoCommand("GetOutboxEventsCommand", func() commands.Command { return &tentant_command.GetOutboxEventsCommand{} })
	RegisterRepoCommand("DeleteOutboxEventsCommand", func() commands.Command { return &tentant_command.DeleteOutboxEventsCommand{} })
	RegisterRepoCommand("CreateTenantInMasterCommand", func() commands.Command { return &tentant_command.CreateTenantInMasterCommand{} })
	RegisterRepoCommand("FindTenantCommand", func() commands.Command { return &tentant_command.FindTenantCommand{} })
	RegisterRepoCommand("PaginateTenantsCommand", func() commands.Command { return &tentant_command.PaginateTenantsCommand{} })
	RegisterRepoCommand("PaginateTenantsWithFilterCommand", func() commands.Command { return &tentant_command.PaginateTenantsWithFilterCommand{} })
	RegisterRepoCommand("MarkTenantActiveCommand", func() commands.Command { return &tentant_command.MarkTenantActiveCommand{} })
	RegisterRepoCommand("MarkTenantInactiveCommand", func() commands.Command { return &tentant_command.MarkTenantInactiveCommand{} })
	RegisterRepoCommand("MarkToDeletionTenantInMasterCommand", func() commands.Command { return &tentant_command.MarkToDeletionTenantInMasterCommand{} })
	RegisterRepoCommand("DeleteTenantInMasterCommand", func() commands.Command { return &tentant_command.DeleteTenantInMasterCommand{} })
	RegisterRepoCommand("AssignToShardTenantInMasterCommand", func() commands.Command { return &tentant_command.AssignToShardTenantInMasterCommand{} })
	RegisterRepoCommand("ResetTenantShardStateCommand", func() commands.Command { return &tentant_command.ResetTenantShardStateCommand{} })
	RegisterRepoCommand("GetDashboardSummaryCommand", func() commands.Command { return &tentant_command.GetDashboardSummaryCommand{} })
	RegisterRepoCommand("UpdateDashboardSummaryCommand", func() commands.Command { return &tentant_command.UpdateDashboardSummaryCommand{} })
	RegisterRepoCommand("GetTenantSummaryInMasterCommand", func() commands.Command { return &tentant_command.GetTenantSummaryInMasterCommand{} })
	RegisterRepoCommand("UpdateTenantSummaryInMasterCommand", func() commands.Command { return &tentant_command.UpdateTenantSummaryInMasterCommand{} })

	// Queue commands
	RegisterRepoCommand("EnqueueCommand", func() commands.Command { return &queue_command.EnqueueCommand{} })
	RegisterRepoCommand("DequeueCommand", func() commands.Command { return &queue_command.DequeueCommand{} })
	RegisterRepoCommand("BatchDequeueCommand", func() commands.Command { return &queue_command.BatchDequeueCommand{} })
	RegisterRepoCommand("BulkDequeueCommand", func() commands.Command { return &queue_command.BulkDequeueCommand{} })
	RegisterRepoCommand("AckMessageCommand", func() commands.Command { return &queue_command.AckMessageCommand{} })
	RegisterRepoCommand("BulkAckMessageCommand", func() commands.Command { return &queue_command.BulkAckMessageCommand{} })
	RegisterRepoCommand("AssertQueueCommand", func() commands.Command { return &queue_command.AssertQueueCommand{} })
	RegisterRepoCommand("FindQueueByIDCommand", func() commands.Command { return &queue_command.FindQueueByIDCommand{} })
	RegisterRepoCommand("FindQueueByIDsCommand", func() commands.Command { return &queue_command.FindQueueByIDsCommand{} })
	RegisterRepoCommand("FindQueueCommand", func() commands.Command { return &queue_command.FindQueueCommand{} })
	RegisterRepoCommand("DeleteQueueCommand", func() commands.Command { return &queue_command.DeleteQueueCommand{} })
	RegisterRepoCommand("MarkLeaseDeliveredCommand", func() commands.Command { return &queue_command.MarkLeaseDeliveredCommand{} })
	RegisterRepoCommand("BulkMarkLeaseDeliveredCommand", func() commands.Command { return &queue_command.BulkMarkLeaseDeliveredCommand{} })
	RegisterRepoCommand("ProcessExpiredLeasesCommand", func() commands.Command { return &queue_command.ProcessExpiredLeasesCommand{} })
	RegisterRepoCommand("MarkQueuesAsDrainCommand", func() commands.Command { return &queue_command.MarkQueuesAsDrainCommand{} })
	RegisterRepoCommand("GetQueueGaugesCommand", func() commands.Command { return &queue_command.GetQueueGaugesCommand{} })
	RegisterRepoCommand("PaginateQueuesCommand", func() commands.Command { return &queue_command.PaginateQueuesCommand{} })
	RegisterRepoCommand("PaginateQueuesWithFilterCommand", func() commands.Command { return &queue_command.PaginateQueuesWithFilterCommand{} })
	RegisterRepoCommand("PaginateQueueMessagesCommand", func() commands.Command { return &queue_command.PaginateQueueMessagesCommand{} })

	// Exchange commands
	RegisterRepoCommand("AssertExchangeCommand", func() commands.Command { return &exchange_command.AssertExchangeCommand{} })
	RegisterRepoCommand("DeleteExchangeCommand", func() commands.Command { return &exchange_command.DeleteExchangeCommand{} })
	RegisterRepoCommand("FindExchangeByIDCommand", func() commands.Command { return &exchange_command.FindExchangeByIDCommand{} })
	RegisterRepoCommand("FindExchangeCommand", func() commands.Command { return &exchange_command.FindExchangeCommand{} })
	RegisterRepoCommand("PaginateExchangesCommand", func() commands.Command { return &exchange_command.PaginateExchangesCommand{} })

	// Binding commands
	RegisterRepoCommand("AssertBindingCommand", func() commands.Command { return &binding_command.AssertBindingCommand{} })
	RegisterRepoCommand("BulkAssertBindingCommand", func() commands.Command { return &binding_command.BulkAssertBindingCommand{} })
	RegisterRepoCommand("ResolveRoutesCommand", func() commands.Command { return &binding_command.ResolveRoutesCommand{} })
	RegisterRepoCommand("ResolveAndFetchQueuesCommand", func() commands.Command { return &binding_command.ResolveAndFetchQueuesCommand{} })
	RegisterRepoCommand("FindBindingCommand", func() commands.Command { return &binding_command.FindBindingCommand{} })
	RegisterRepoCommand("DeleteBindingCommand", func() commands.Command { return &binding_command.DeleteBindingCommand{} })
	RegisterRepoCommand("PaginateBindingsCommand", func() commands.Command { return &binding_command.PaginateBindingsCommand{} })
	RegisterRepoCommand("PaginateByExchangeBindingsCommand", func() commands.Command { return &binding_command.PaginateByExchangeBindingsCommand{} })

	// User commands
	RegisterRepoCommand("CreateUserCommand", func() commands.Command { return &user_command.CreateUserCommand{} })
	RegisterRepoCommand("DeleteUserCommand", func() commands.Command { return &user_command.DeleteUserCommand{} })
	RegisterRepoCommand("GetUsersCommand", func() commands.Command { return &user_command.GetUsersCommand{} })
	RegisterRepoCommand("UpdateUserCommand", func() commands.Command { return &user_command.UpdateUserCommand{} })

	// VNamespace commands
	RegisterRepoCommand("PaginateVNamespacesCommand", func() commands.Command { return &vnamespace_command.PaginateVNamespacesCommand{} })
	RegisterRepoCommand("PaginateVNamespacesWithFilterCommand", func() commands.Command { return &vnamespace_command.PaginateVNamespacesWithFilterCommand{} })

	// Header commands
	RegisterRepoCommand("ListHeadersCommand", func() commands.Command { return &header_command.ListHeadersCommand{} })

	// Job Worker commands
	RegisterRepoCommand("UpsertJobWorkerCommand", func() commands.Command { return &jobworker_command.UpsertJobWorkerCommand{} })
	RegisterRepoCommand("PaginateJobWorkersCommand", func() commands.Command { return &jobworker_command.PaginateJobWorkersCommand{} })

	// Metrics commands
	RegisterRepoCommand("SaveMetricsBucketsCommand", func() commands.Command { return &metrics_command.SaveMetricsBucketsCommand{} })
	RegisterRepoCommand("QueryMetricsRangeCommand", func() commands.Command { return &metrics_command.QueryMetricsRangeCommand{} })
	RegisterRepoCommand("DeleteExpiredMetricsCommand", func() commands.Command { return &metrics_command.DeleteExpiredMetricsCommand{} })
	RegisterRepoCommand("DownsampleMetricsCommand", func() commands.Command { return &metrics_command.DownsampleMetricsCommand{} })

	// General commands
	RegisterRepoCommand("CreateColumnFamilyCommand", func() commands.Command { return &CreateColumnFamilyCommand{} })
	RegisterRepoCommand("DeleteColumnFamilyCommand", func() commands.Command { return &DeleteColumnFamilyCommand{} })
	RegisterRepoCommand("DeleteColumnFamilySectorCommand", func() commands.Command { return &DeleteColumnFamilySectorCommand{} })
}
