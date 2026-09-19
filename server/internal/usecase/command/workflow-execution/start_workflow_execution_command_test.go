package workflow_execution_test

import (
	"os"
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	workflowExecCommand "deadalus-orch/server/internal/usecase/command/workflow-execution"
	"deadalus-orch/shared/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPebbleStore(t *testing.T) db.KVStore {
	tempDir, err := os.MkdirTemp("", "workflow_exec_test_*")
	require.NoError(t, err)

	cfNames := []string{"default", db.AdminFC}
	store, err := db.CreatePebbleStore(tempDir, cfNames, []string{})
	require.NoError(t, err)

	t.Cleanup(func() {
		store.Close()
		os.RemoveAll(tempDir)
	})
	return store
}

func TestStartWorkflowExecutionCommand_DesignErrorsValidation(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}
	now := time.Now()

	// Create workflow definition with design errors
	defRepo, err := db.NewWorkflowDefinitionRepository(uow, idFactory, "default", "default")
	require.NoError(t, err)

	def := &models.WorkflowDefinition{
		ID:                  "def_with_errors",
		Code:                "DEF_ERR_01",
		VNamespace:          "default",
		Name:                "Workflow With Errors",
		Version:             1,
		Payload:             []byte(`<?xml version="1.0"?><bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="D1"></bpmn:definitions>`),
		IsActive:            true,
		HasDesignErrors:     true,
		DesignErrorMessages: []string{"Process must contain at least one Start Event", "Process must contain at least one End Event"},
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	_, err = defRepo.CreateWorkflowDefinition(def, now)
	require.NoError(t, err)
	require.NoError(t, uow.Commit())

	execUow := db.NewUnitOfWork(store, nil)
	cmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		WorkflowDefinitionID: "def_with_errors",
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}

	res := cmd.Execute(execUow, now)
	assert.NotEmpty(t, res.Error)
	assert.Contains(t, res.Error, "cannot execute workflow definition: workflow has design errors")
}
