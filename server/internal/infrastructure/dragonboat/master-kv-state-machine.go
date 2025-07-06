package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	commands "deadalus-orch/server/internal/usecase/command"
	"errors"
	"time"

	"github.com/lni/dragonboat/v4/statemachine"
)

type MasterKVDBStateMachine struct {
}

func (r *MasterKVDBStateMachine) OpenDB(dbPath string) (db.KVStore, error) {
	return db.OpenMasterDB(dbPath)
}

func (r *MasterKVDBStateMachine) Lookup(input any, uow *db.UnitOfWork, now time.Time) (interface{}, error) {

	loginCmd, ok := input.(commands.LoginCommand)
	if ok {
		return loginCmd.Execute(uow, now)
	}

	checkSessionExistsCommand, ok := input.(commands.CheckSessionExistsCommand)
	if ok {
		return checkSessionExistsCommand.Execute(uow, now)
	}

	return nil, errors.New("invalid command type")
}

func (r *MasterKVDBStateMachine) Update(cmd any, uow *db.UnitOfWork, now time.Time) ([]byte, error) {
	bootstrapRootUserCmd, ok := cmd.(commands.BootstrapRootUserCommand)
	if ok {
		return bootstrapRootUserCmd.Execute(uow, now)
	}

	registerSessionCommand, ok := cmd.(commands.RegisterSessionCommand)
	if ok {
		return registerSessionCommand.Execute(uow, now)
	}

	return nil, errors.New("invalid command type")
}

func NewMasterKVStateMachine(pathProvider db.PathProvider, port int) func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	return func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
		return NewKVStateMachine(clusterID, nodeID, port, &MasterKVDBStateMachine{}, KVBaseStateMachineConfig{
			TTLInternalError: config.GlobalConfiguration.TTLInternalError,
			PathProvider:     pathProvider,
		})
	}

}
