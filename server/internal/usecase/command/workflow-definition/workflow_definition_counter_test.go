package workflow_definition_test

import (
	"os"
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	workflowDefCommand "deadalus-orch/server/internal/usecase/command/workflow-definition"
	tenantCommand "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCounterPebbleStore(t *testing.T) db.KVStore {
	config.GlobalConfiguration = &config.Config{
		MaxHeaders: 100,
	}

	tempDir, err := os.MkdirTemp("", "wf_counter_test_*")
	require.NoError(t, err)

	cfNames := []string{"default", "cf0", db.AdminFC}
	store, err := db.CreatePebbleStore(tempDir, cfNames, []string{})
	require.NoError(t, err)

	t.Cleanup(func() {
		store.Close()
		os.RemoveAll(tempDir)
	})
	return store
}

func TestWorkflowDefinitionCounter_CreateAndDelete(t *testing.T) {
	store := newTestCounterPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}
	now := time.Now()

	tenantID := "tenant-test-1"
	cf := "cf0"
	cfs := tenantID

	// 1. Create first workflow definition
	createCmd1 := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:                 "wf-1",
			Code:               "WF_CODE_1",
			Name:               "Workflow 1",
			VNamespace:         "default",
			Payload:            []byte(`{}`),
			PayloadFormat:      models.WorkflowPayloadFormatJSON,
			MaxDurationSeconds: 60,
			IsActive:           true,
			Scope:              models.WorkflowScopeTenant,
			TenantID:           tenantID,
		},
		CF:  cf,
		CFS: cfs,
	}

	res1 := createCmd1.Execute(uow, now)
	require.Empty(t, res1.Error)
	require.NoError(t, uow.Commit())

	// Verify TenantSummary WorkflowsCount is 1
	uowRead := db.NewUnitOfWork(store, nil)
	summaryRepo, err := db.NewTenantSummaryRepository(uowRead, idFactory)
	require.NoError(t, err)

	summary, err := summaryRepo.GetTenantSummaryById(tenantID, now)
	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, 1, summary.WorkflowsCount)

	// 2. Create second workflow definition
	uow2 := db.NewUnitOfWork(store, nil)
	createCmd2 := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:                 "wf-2",
			Code:               "WF_CODE_2",
			Name:               "Workflow 2",
			VNamespace:         "default",
			Payload:            []byte(`{}`),
			PayloadFormat:      models.WorkflowPayloadFormatJSON,
			MaxDurationSeconds: 60,
			IsActive:           true,
			Scope:              models.WorkflowScopeTenant,
			TenantID:           tenantID,
		},
		CF:  cf,
		CFS: cfs,
	}

	res2 := createCmd2.Execute(uow2, now)
	require.Empty(t, res2.Error)
	require.NoError(t, uow2.Commit())

	// Verify TenantSummary WorkflowsCount is now 2
	uowRead2 := db.NewUnitOfWork(store, nil)
	summaryRepo2, err := db.NewTenantSummaryRepository(uowRead2, idFactory)
	require.NoError(t, err)

	summary2, err := summaryRepo2.GetTenantSummaryById(tenantID, now)
	require.NoError(t, err)
	require.NotNil(t, summary2)
	assert.Equal(t, 2, summary2.WorkflowsCount)

	// 3. Delete first workflow definition
	uow3 := db.NewUnitOfWork(store, nil)
	deleteCmd := &workflowDefCommand.DeleteWorkflowDefinitionCommand{
		WorkflowID: "wf-1",
		CF:         cf,
		CFS:        cfs,
	}

	delRes := deleteCmd.Execute(uow3, now)
	require.Empty(t, delRes.Error)
	require.NoError(t, uow3.Commit())

	// Verify TenantSummary WorkflowsCount is decremented back to 1
	uowRead3 := db.NewUnitOfWork(store, nil)
	summaryRepo3, err := db.NewTenantSummaryRepository(uowRead3, idFactory)
	require.NoError(t, err)

	summary3, err := summaryRepo3.GetTenantSummaryById(tenantID, now)
	require.NoError(t, err)
	require.NotNil(t, summary3)
	assert.Equal(t, 1, summary3.WorkflowsCount)
}

func TestTenantSummaryInMaster_WorkflowsCount(t *testing.T) {
	store := newTestCounterPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}
	now := time.Now()

	// Create TenantInMaster record
	masterRepo, err := db.NewTenantInMasterRepository(uow, idFactory)
	require.NoError(t, err)

	tenantRecord := &models.TenantInMaster{
		ID:        "t-100",
		Name:      "Test Tenant",
		Code:      "TT100",
		Status:    models.Assigned,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err = masterRepo.CreateTenantInMaster(tenantRecord, now)
	require.NoError(t, err)
	require.NoError(t, uow.Commit())

	// Update summary with WorkflowsCount = 5
	uowUpdate := db.NewUnitOfWork(store, nil)
	updateCmd := &tenantCommand.UpdateTenantSummaryInMasterCommand{
		TenantSummaries: []models.TenantSummary{
			{
				ID:             "t-100",
				WorkflowsCount: 5,
				QueuesCount:    4,
				MessagesCount:  10,
			},
		},
	}
	res := updateCmd.Execute(uowUpdate, now)
	require.Empty(t, res.Error)
	require.NoError(t, uowUpdate.Commit())

	// Retrieve tenant from master and verify WorkflowsCount
	uowGet := db.NewUnitOfWork(store, nil)
	getCmd := &tenantCommand.GetTenantSummaryInMasterCommand{
		TenantCode: "TT100",
	}
	getRes := getCmd.Execute(uowGet, now)
	require.Empty(t, getRes.Error)

	summaryResult, ok := getRes.Result.(models.TenantSummary)
	require.True(t, ok)
	assert.Equal(t, 5, summaryResult.WorkflowsCount)
	assert.Equal(t, 4, summaryResult.QueuesCount)
	assert.Equal(t, 10, summaryResult.MessagesCount)
}
