package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
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

func (r *TenantKVBaseStateMachine) Lookup(cmd any, uow *db.UnitOfWork, now time.Time) (interface{}, error) {
	return nil, nil
}

func (r *TenantKVBaseStateMachine) Update(cmd any, uow *db.UnitOfWork, now time.Time) ([]byte, error) {
	return nil, nil
}

func NewTenantKVStateMachine(pathProvider db.PathProvider, port int) func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	return func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
		return NewKVStateMachine(clusterID, nodeID, port, &TenantKVBaseStateMachine{}, KVBaseStateMachineConfig{
			TTLInternalError: config.GlobalConfiguration.TTLInternalError,
			PathProvider:     pathProvider,
		})
	}

}
