package workflow_execution_test

import (
	"os"
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/crypto"
	workflowDefCommand "deadalus-orch/server/internal/usecase/command/workflow-definition"
	workflowExecCommand "deadalus-orch/server/internal/usecase/command/workflow-execution"
	"deadalus-orch/shared/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvanceTokenCommand_ExpressionEvaluation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "expr_eval_test_*")
	require.NoError(t, err)

	cfNames := []string{"default", db.AdminFC}
	store, err := db.CreatePebbleStore(tempDir, cfNames, []string{})
	require.NoError(t, err)
	t.Cleanup(func() {
		store.Close()
		os.RemoveAll(tempDir)
	})

	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()

	bpmnXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_1">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="Start_1">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:serviceTask id="Task_HTTP" name="Call HTTP Endpoint">
      <bpmn:extensionElements>
        <camunda:property name="url" value="https://${host}/api/v1/users/${userId}" />
        <camunda:property name="greeting" value="Hello ${user.firstName} ${user.lastName}!" />
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_1</bpmn:incoming>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="Start_1" targetRef="Task_HTTP" />
  </bpmn:process>
</bpmn:definitions>`

	defID := "def_expr_test"
	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:         defID,
			Code:       "CODE_EXPR_TEST",
			VNamespace: "default",
			Name:       "Expression Test",
			Version:    1,
			Payload:    []byte(bpmnXML),
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		ExecQueue: models.Queue{ID: "q_exec_" + defID},
		ActQueue:  models.Queue{ID: "q_act_" + defID},
		CF:        "default",
		CFS:       "default",
	}
	resDef := defCmd.Execute(uow, now)
	require.Empty(t, resDef.Error)
	require.NoError(t, uow.Commit())

	// Start Workflow Execution with input variables
	startUow := db.NewUnitOfWork(store, nil)
	inputVars := map[string]interface{}{
		"host":   "api.daedalus.io",
		"userId": "usr_998234",
		"user": map[string]interface{}{
			"firstName": "Angel",
			"lastName":  "Guerrero",
		},
	}

	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		ExecutionID:          "exec_expr_01",
		WorkflowDefinitionID: defID,
		Input:                inputVars,
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	resStart := startCmd.Execute(startUow, now)
	require.Empty(t, resStart.Error)
	require.NoError(t, startUow.Commit())

	// Advance Token from Start Event to Service Task
	tokenUow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}
	tokenRepo, err := db.NewExecutionTokenRepository(tokenUow, idFactory, "default", "default")
	require.NoError(t, err)
	activeTokens, err := tokenRepo.GetActiveTokensByExecutionID("exec_expr_01", now)
	require.NoError(t, err)
	require.NotEmpty(t, activeTokens)

	advUow := db.NewUnitOfWork(store, nil)
	advCmd := &workflowExecCommand.AdvanceTokenCommand{
		ExecutionID: "exec_expr_01",
		TokenID:     activeTokens[0].ID,
		CF:          "default",
		CFS:         "default",
	}
	resAdv := advCmd.Execute(advUow, now)
	require.Empty(t, resAdv.Error)
	require.NoError(t, advUow.Commit())

	// Verify Created Workflow Job Payload Has Resolved Expressions
	checkUow := db.NewUnitOfWork(store, nil)
	jobRepo, err := db.NewWorkflowJobRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)

	jobs, err := jobRepo.GetJobsByExecutionID("exec_expr_01", now)
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	jobPayload := jobs[0].Input
	assert.Equal(t, "https://api.daedalus.io/api/v1/users/usr_998234", jobPayload["url"])
	assert.Equal(t, "Hello Angel Guerrero!", jobPayload["greeting"])
}

func TestAdvanceTokenCommand_ConfigAndSecretEvaluation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_secret_test_*")
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

	// 1. Setup Global Config & Secret in AdminFC
	globalUow := db.NewUnitOfWork(store, nil)
	globalGroupRepo, err := db.NewEnvGroupRepository(globalUow, idFactory, db.AdminFC, db.AdminFCSector)
	require.NoError(t, err)
	globalVarRepo, err := db.NewEnvVarRepository(globalUow, idFactory, db.AdminFC, db.AdminFCSector)
	require.NoError(t, err)

	gCfgID, err := globalGroupRepo.CreateEnvGroup(&models.EnvGroup{
		ID:         "grp_g_cfg",
		Code:       "global_cfg",
		Name:       "Global Config",
		Type:       models.EnvGroupTypeConfig,
		Scope:      models.EnvGroupScopeGlobal,
		VNamespace: "default",
	}, now)
	require.NoError(t, err)

	_, err = globalVarRepo.CreateEnvVar(&models.EnvVar{
		ID:          "v1",
		GroupID:     gCfgID,
		GroupIDComp: gCfgID,
		Key:         "SERVER_HOST",
		Value:       "global.domain.com",
	}, now)
	require.NoError(t, err)

	gSecID, err := globalGroupRepo.CreateEnvGroup(&models.EnvGroup{
		ID:         "grp_g_sec",
		Code:       "global_sec",
		Name:       "Global Secret",
		Type:       models.EnvGroupTypeSecret,
		Scope:      models.EnvGroupScopeGlobal,
		VNamespace: "default",
	}, now)
	require.NoError(t, err)

	encGlobalKey, err := crypto.Encrypt("master_key_999")
	require.NoError(t, err)
	_, err = globalVarRepo.CreateEnvVar(&models.EnvVar{
		ID:          "v2",
		GroupID:     gSecID,
		GroupIDComp: gSecID,
		Key:         "API_KEY",
		Value:       encGlobalKey,
	}, now)
	require.NoError(t, err)
	require.NoError(t, globalUow.Commit())

	// 2. Setup Tenant Config & Secret in Tenant CF ("default")
	tenantUow := db.NewUnitOfWork(store, nil)
	tenantGroupRepo, err := db.NewEnvGroupRepository(tenantUow, idFactory, "default", "default")
	require.NoError(t, err)
	tenantVarRepo, err := db.NewEnvVarRepository(tenantUow, idFactory, "default", "default")
	require.NoError(t, err)

	tCfgID, err := tenantGroupRepo.CreateEnvGroup(&models.EnvGroup{
		ID:         "grp_t_cfg",
		Code:       "tenant_cfg",
		Name:       "Tenant Config",
		Type:       models.EnvGroupTypeConfig,
		Scope:      models.EnvGroupScopeTenant,
		VNamespace: "default",
	}, now)
	require.NoError(t, err)

	_, err = tenantVarRepo.CreateEnvVar(&models.EnvVar{
		ID:          "v3",
		GroupID:     tCfgID,
		GroupIDComp: tCfgID,
		Key:         "DB_NAME",
		Value:       "tenant_db_1",
	}, now)
	require.NoError(t, err)

	tSecID, err := tenantGroupRepo.CreateEnvGroup(&models.EnvGroup{
		ID:         "grp_t_sec",
		Code:       "tenant_sec_code",
		Name:       "Tenant Secret Group Name",
		Type:       models.EnvGroupTypeSecret,
		Scope:      models.EnvGroupScopeTenant,
		VNamespace: "default",
	}, now)
	require.NoError(t, err)

	encTenantPass, err := crypto.Encrypt("super_secret_pwd")
	require.NoError(t, err)
	_, err = tenantVarRepo.CreateEnvVar(&models.EnvVar{
		ID:          "v4",
		GroupID:     tSecID,
		GroupIDComp: tSecID,
		Key:         "DB_PASS",
		Value:       encTenantPass,
	}, now)
	require.NoError(t, err)
	require.NoError(t, tenantUow.Commit())

	// 3. Define BPMN workflow using ${config...} and ${secret...}
	bpmnXML := `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" id="Definitions_2">
  <bpmn:process id="Process_2" isExecutable="true">
    <bpmn:startEvent id="Start_1">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:serviceTask id="Task_Secret" name="Call External Service">
      <bpmn:extensionElements>
        <camunda:property name="globalHost" value="${config.global['global_cfg']['SERVER_HOST']}" />
        <camunda:property name="globalKey" value="${secret.global['global_sec']['API_KEY']}" />
        <camunda:property name="tenantDb" value="${config.tenant['tenant_cfg']['DB_NAME']}" />
        <camunda:property name="tenantPass" value="${secret.tenant['Tenant Secret Group Name']['DB_PASS']}" />
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_1</bpmn:incoming>
    </bpmn:serviceTask>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="Start_1" targetRef="Task_Secret" />
  </bpmn:process>
</bpmn:definitions>`

	defID := "def_cfg_sec_test"
	defCmd := &workflowDefCommand.CreateWorkflowDefinitionCommand{
		WorkflowDefinition: models.WorkflowDefinition{
			ID:         defID,
			Code:       "CODE_CFG_SEC_TEST",
			VNamespace: "default",
			Name:       "Config Secret Test",
			Version:    1,
			Payload:    []byte(bpmnXML),
			IsActive:   true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		ExecQueue: models.Queue{ID: "q_exec_" + defID},
		ActQueue:  models.Queue{ID: "q_act_" + defID},
		CF:        "default",
		CFS:       "default",
	}
	uowDef := db.NewUnitOfWork(store, nil)
	resDef := defCmd.Execute(uowDef, now)
	require.Empty(t, resDef.Error)
	require.NoError(t, uowDef.Commit())

	// Start Workflow Execution
	startUow := db.NewUnitOfWork(store, nil)
	startCmd := &workflowExecCommand.StartWorkflowExecutionCommand{
		ExecutionID:          "exec_cfg_sec_01",
		WorkflowDefinitionID: defID,
		VNamespace:           "default",
		CF:                   "default",
		CFS:                  "default",
	}
	resStart := startCmd.Execute(startUow, now)
	require.Empty(t, resStart.Error)
	require.NoError(t, startUow.Commit())

	// Advance Token
	tokenUow := db.NewUnitOfWork(store, nil)
	tokenRepo, err := db.NewExecutionTokenRepository(tokenUow, idFactory, "default", "default")
	require.NoError(t, err)
	activeTokens, err := tokenRepo.GetActiveTokensByExecutionID("exec_cfg_sec_01", now)
	require.NoError(t, err)
	require.NotEmpty(t, activeTokens)

	advUow := db.NewUnitOfWork(store, nil)
	advCmd := &workflowExecCommand.AdvanceTokenCommand{
		ExecutionID: "exec_cfg_sec_01",
		TokenID:     activeTokens[0].ID,
		CF:          "default",
		CFS:         "default",
	}
	resAdv := advCmd.Execute(advUow, now)
	require.Empty(t, resAdv.Error)
	require.NoError(t, advUow.Commit())

	// Verify Job Payload
	checkUow := db.NewUnitOfWork(store, nil)
	jobRepo, err := db.NewWorkflowJobRepository(checkUow, idFactory, "default", "default")
	require.NoError(t, err)

	jobs, err := jobRepo.GetJobsByExecutionID("exec_cfg_sec_01", now)
	require.NoError(t, err)
	require.NotEmpty(t, jobs)

	jobPayload := jobs[0].Input
	assert.Equal(t, "global.domain.com", jobPayload["globalHost"])
	assert.Equal(t, "master_key_999", jobPayload["globalKey"])
	assert.Equal(t, "tenant_db_1", jobPayload["tenantDb"])
	assert.Equal(t, "super_secret_pwd", jobPayload["tenantPass"])
}

