package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	commands "deadalus-orch/server/internal/usecase/command"
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
	commandResult := &commands.CommandResult{}
	commandResult.Error = "invalid command type"
	return *commandResult
}

func (r *TenantKVBaseStateMachine) Update(cmd any, uow *db.UnitOfWork, now time.Time) commands.CommandResult {
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
