package workflow_execution_test

import (
	"os"
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	workflowDefCommand "deadalus-orch/server/internal/usecase/command/workflow-definition"
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

func TestStartWorkflowExecutionCommand_BusinessKeyReuse(t *testing.T) {
	store := newTestPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}
	now := time.Now()

	validBPMN := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_1">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="Start_1" />
  </bpmn:process>
</bpmn:definitions>`

	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:         "def_bk_test",
			Code:       "DEF_BK_01",
			VNamespace: "default",
			Name:       "Business Key Test",
			Version:    1,
			Payload:    []byte(validBPMN),
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		ExecQueue: models.Queue{ID: "q_exec_def_bk_test"},
		ActQueue:  models.Queue{ID: "q_act_def_bk_test"},
		CF:        "default",
		CFS:       "default",
	}
	resDef := defCmd.Execute(uow, now)
	require.Empty(t, resDef.Error)
	require.NoError(t, uow.Commit())

	// 1. Create initial running execution with business key "KEY-100"
	execUow1 := db.NewUnitOfWork(store, nil)
	cmd1 := &workflowExecCommand.StartWorkflowExecutionCommand{
		ExecutionID:          "exec_100",
		WorkflowDefinitionID: "def_bk_test",
		ExecutionKey:         "KEY-100",
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	res1 := cmd1.Execute(execUow1, now)
	require.Empty(t, res1.Error, "First execution with KEY-100 should succeed")
	require.NoError(t, execUow1.Commit())

	// 2. Try starting a second execution with KEY-100 while the first is still running -> should fail
	execUow2 := db.NewUnitOfWork(store, nil)
	cmd2 := &workflowExecCommand.StartWorkflowExecutionCommand{
		ExecutionID:          "exec_101",
		WorkflowDefinitionID: "def_bk_test",
		ExecutionKey:         "KEY-100",
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	res2 := cmd2.Execute(execUow2, now)
	assert.NotEmpty(t, res2.Error)
	assert.Contains(t, res2.Error, "is already in execution")

	// 3. Mark the first execution as completed
	updateUow := db.NewUnitOfWork(store, nil)
	execRepo, err := db.NewWorkflowExecutionRepository(updateUow, idFactory, "default", "default")
	require.NoError(t, err)
	exec1, err := execRepo.GetWorkflowExecutionByID("exec_100", now)
	require.NoError(t, err)
	require.NotNil(t, exec1)
	exec1.Status = models.WorkflowExecutionStatusCompleted
	_, err = execRepo.UpdateWorkflowExecution(exec1, now)
	require.NoError(t, err)
	require.NoError(t, updateUow.Commit())

	// 4. Start a new execution with KEY-100 after the previous execution finished -> should succeed!
	execUow3 := db.NewUnitOfWork(store, nil)
	cmd3 := &workflowExecCommand.StartWorkflowExecutionCommand{
		ExecutionID:          "exec_102",
		WorkflowDefinitionID: "def_bk_test",
		ExecutionKey:         "KEY-100",
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	res3 := cmd3.Execute(execUow3, now)
	assert.Empty(t, res3.Error, "Should allow reusing business key KEY-100 when previous execution is completed")
	require.NoError(t, execUow3.Commit())
}
