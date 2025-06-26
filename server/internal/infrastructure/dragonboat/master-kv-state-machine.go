package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	usecasecommand "deadalus-orch/server/internal/usecase/command" // Added import
	"fmt"                                                         // Added import
	"time"

	"github.com/lni/dragonboat/v4/statemachine"
)

type MasterKVDBStateMachine struct {
}

func (r *MasterKVDBStateMachine) OpenDB(dbPath string) (db.KVStore, error) {
	return db.OpenMasterDB(dbPath)
}

func (r *MasterKVDBStateMachine) Lookup(input any, uow *db.UnitOfWork, now time.Time) (interface{}, error) {
	queryCmd, ok := input.(*Query_Command)
	if !ok {
		return nil, fmt.Errorf("invalid command type: expected *Query_Command, got %T", input)
	}

	// The Query_Command.Command field holds the actual command payload.
	// This payload should implement our usecasecommand.Command interface.
	executableCmd, ok := queryCmd.Command.(usecasecommand.Command)
	if !ok {
		return nil, fmt.Errorf("command within Query_Command does not implement usecasecommand.Command interface")
	}

	// Execute the command
	return executableCmd.Execute(uow, now)
}

func (r *MasterKVDBStateMachine) Update(cmd any, uow *db.UnitOfWork, now time.Time) ([]byte, error) {
	return nil, nil
}

func NewMasterKVStateMachine(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	return NewKVStateMachine(clusterID, nodeID, &MasterKVDBStateMachine{}, KVBaseStateMachineConfig{
		TTLInternalError: config.GlobalConfiguration.TTLInternalError,
	})
}
