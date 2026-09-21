package workflowexecution

import (
	"net/http"
	"strconv"

	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"

	"github.com/gin-gonic/gin"
)

type WorkflowExecutionController struct {
	Config              *common.ServerConfing
	WorkflowExecutionBO *bo.WorkflowExecutionBO
}

func NewWorkflowExecutionController(config *common.ServerConfing) *WorkflowExecutionController {
	return &WorkflowExecutionController{
		Config:              config,
		WorkflowExecutionBO: bo.NewWorkflowExecutionBO(config),
	}
}

type startExecutionRequest struct {
	WorkflowDefinitionID string                     `json:"workflowDefinitionId"`
	ExecutionKey         string                     `json:"executionKey"`
	OnVersionChange      models.VersionChangePolicy `json:"onVersionChange"`
	Input                map[string]interface{}     `json:"input"`
	VNamespace           string                     `json:"vnamespace"`
}

type completeJobRequest struct {
	WorkerID   string                 `json:"workerId"`
	OutputData map[string]interface{} `json:"outputData"`
	Error      string                 `json:"error"`
}

// --- GLOBAL HANDLERS ---

func (ctrl *WorkflowExecutionController) StartGlobalExecutionHandler(c *gin.Context) {
	var req startExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	exec, err := ctrl.WorkflowExecutionBO.StartExecution(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		req.WorkflowDefinitionID,
		req.ExecutionKey,
		req.OnVersionChange,
		req.Input,
		req.VNamespace,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, exec)
}

func (ctrl *WorkflowExecutionController) GetGlobalExecutionHandler(c *gin.Context) {
	id := c.Param("id")
	exec, err := ctrl.WorkflowExecutionBO.GetExecution(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		id,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if exec == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow execution not found"})
		return
	}

	c.JSON(http.StatusOK, exec)
}

func (ctrl *WorkflowExecutionController) ListGlobalExecutionsHandler(c *gin.Context) {
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	cursor := c.Query("cursor")
	vnamespace := c.Query("vnamespace")
	workflowDefinitionID := c.Query("workflowDefinitionId")
	status := c.Query("status")

	res, err := ctrl.WorkflowExecutionBO.ListExecutions(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		vnamespace,
		workflowDefinitionID,
		status,
		pageSize,
		cursor,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

func (ctrl *WorkflowExecutionController) CompleteGlobalJobHandler(c *gin.Context) {
	jobID := c.Param("jobId")
	var req completeJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	res, err := ctrl.WorkflowExecutionBO.CompleteJob(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		jobID,
		req.WorkerID,
		req.OutputData,
		req.Error,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}

// --- TENANT HANDLERS ---

func (ctrl *WorkflowExecutionController) StartTenantExecutionHandler(c *gin.Context) {
	var req startExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	exec, err := ctrl.WorkflowExecutionBO.StartExecution(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		req.WorkflowDefinitionID,
		req.ExecutionKey,
		req.OnVersionChange,
		req.Input,
		req.VNamespace,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, exec)
}

func (ctrl *WorkflowExecutionController) GetTenantExecutionHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	id := c.Param("id")

	exec, err := ctrl.WorkflowExecutionBO.GetExecution(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		id,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if exec == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow execution not found"})
		return
	}

	c.JSON(http.StatusOK, exec)
}

func (ctrl *WorkflowExecutionController) ListTenantExecutionsHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	cursor := c.Query("cursor")
	vnamespace := c.Query("vnamespace")
	workflowDefinitionID := c.Query("workflowDefinitionId")
	status := c.Query("status")

	res, err := ctrl.WorkflowExecutionBO.ListExecutions(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		vnamespace,
		workflowDefinitionID,
		status,
		pageSize,
		cursor,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res)
}
