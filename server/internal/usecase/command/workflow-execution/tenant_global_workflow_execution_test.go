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

func TestTenantGlobalWorkflowExecution_Isolation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tenant_global_exec_test_*")
	require.NoError(t, err)

	cfNames := []string{"default", db.AdminFC}
	store, err := db.CreatePebbleStore(tempDir, cfNames, []string{})
	require.NoError(t, err)

	t.Cleanup(func() {
		store.Close()
		os.RemoveAll(tempDir)
	})

	idFactory := &db.DeterministicIDGeneratorFactory{}
	now := time.Now()

	// 1. Create a Global Workflow Definition in db.AdminFC, db.AdminFCSector
	globalUow := db.NewUnitOfWork(store, nil)
	bpmnXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" id="Definitions_1">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="Start_1">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:endEvent id="End_1">
      <bpmn:incoming>Flow_1</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="Start_1" targetRef="End_1" />
  </bpmn:process>
</bpmn:definitions>`

	globalDefCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:                 "global_wf_01",
			Code:               "GLOBAL_WF_01",
			Name:               "Global Workflow",
			VNamespace:         "default",
			Version:            1,
			Scope:              models.WorkflowScopeGlobal,
			Payload:            []byte(bpmnXML),
			PayloadFormat:      models.WorkflowPayloadFormatJSON,
			MaxDurationSeconds: 3600,
			IsActive:           true,
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		ExecQueue: models.Queue{ID: "q_exec_global_wf_01"},
		ActQueue:  models.Queue{ID: "q_act_global_wf_01"},
		CF:        db.AdminFC,
		CFS:       db.AdminFCSector,
	}

	res := globalDefCmd.Execute(globalUow, now)
	require.Empty(t, res.Error)
	require.NoError(t, globalUow.Commit())

	// Verify global workflow definition exists in AdminFC
	checkGlobalUow := db.NewUnitOfWork(store, nil)
	globalDefRepo, err := db.NewWorkflowDefinitionRepository(checkGlobalUow, idFactory, db.AdminFC, db.AdminFCSector)
	require.NoError(t, err)
	globalWf, err := globalDefRepo.GetWorkflowDefinitionByID("global_wf_01", now)
	require.NoError(t, err)
	require.NotNil(t, globalWf)
	assert.Equal(t, "GLOBAL_WF_01", globalWf.Code)

	// 2. Start execution of this global workflow in a Tenant sector: CF="default", CFS="tenant_sector_a"
	tenantCF := "default"
	tenantCFS := "tenant_sector_a"

	tenantUow := db.NewUnitOfWork(store, nil)
	startExecCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		ExecutionID:          "tenant_exec_01",
		InitialTokenID:       "tenant_token_01",
		WorkflowDefinitionID: "global_wf_01",
		ExecutionKey:         "order-12345",
		OnVersionChange:      models.VersionChangePolicyContinue,
		VNamespace:           "default",
		CF:                   tenantCF,
		CFS:                  tenantCFS,
	}

	execRes := startExecCmd.Execute(tenantUow, now)
	require.Empty(t, execRes.Error, "Starting execution of global workflow in tenant sector must succeed")
	require.NoError(t, tenantUow.Commit())

	createdExec, ok := execRes.Result.(*models.WorkflowExecution)
	require.True(t, ok)
	assert.Equal(t, "tenant_exec_01", createdExec.ID)
	assert.Equal(t, "global_wf_01", createdExec.WorkflowDefinitionID)
	assert.Equal(t, models.WorkflowExecutionStatusRunning, createdExec.Status)

	// 3. Verify tenant auto-provisioned queues
	verifyTenantUow := db.NewUnitOfWork(store, nil)
	tenantQueueRepo, err := db.NewQueueRepository(verifyTenantUow, idFactory, tenantCF, tenantCFS)
	require.NoError(t, err)
	tenantQueues, err := tenantQueueRepo.GetQueuesByWorkflowDefinitionID("global_wf_01", now)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tenantQueues), 1, "Tenant should have auto-provisioned execution and activity queues")

	// 4. Verify ISOLATION:
	// a) Listing executions in Global sector (AdminFC / AdminFCSector) must return 0 executions
	globalExecRepo, err := db.NewWorkflowExecutionRepository(checkGlobalUow, idFactory, db.AdminFC, db.AdminFCSector)
	require.NoError(t, err)
	globalExecList, err := globalExecRepo.ListWorkflowExecutions("default", "global_wf_01", "", 50, "", now)
	require.NoError(t, err)
	assert.Empty(t, globalExecList.Entities, "Global context must NOT see tenant executions of global workflow")

	// b) Listing executions in Tenant sector must return the execution
	tenantExecRepo, err := db.NewWorkflowExecutionRepository(verifyTenantUow, idFactory, tenantCF, tenantCFS)
	require.NoError(t, err)
	tenantExecList, err := tenantExecRepo.ListWorkflowExecutions("default", "global_wf_01", "", 50, "", now)
	require.NoError(t, err)
	require.Len(t, tenantExecList.Entities, 1, "Tenant context must see its execution of global workflow")
	assert.Equal(t, "tenant_exec_01", tenantExecList.Entities[0].ID)

	// 5. Advance token within the tenant partition
	advanceUow := db.NewUnitOfWork(store, nil)
	advanceCmd := &workflowExecCommand.AdvanceTokenCommand{
		ExecutionID: "tenant_exec_01",
		TokenID:     "tenant_token_01",
		CF:          tenantCF,
		CFS:         tenantCFS,
	}
	advRes := advanceCmd.Execute(advanceUow, now)
	require.Empty(t, advRes.Error, "AdvanceTokenCommand must succeed using fallback to global workflow definition")
	require.NoError(t, advanceUow.Commit())

	updatedExec, ok := advRes.Result.(*models.WorkflowExecution)
	require.True(t, ok)
	assert.Equal(t, models.WorkflowExecutionStatusCompleted, updatedExec.Status)
}
