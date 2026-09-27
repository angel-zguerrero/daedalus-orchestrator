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

func (ctrl *WorkflowExecutionController) SubmitEventInputHandler(c *gin.Context) {
	var rawBody map[string]interface{}
	if err := c.ShouldBindJSON(&rawBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	waitingEventID := c.Param("waitingEventId")
	if waitingEventID == "" {
		waitingEventID = c.Param("id")
	}
	if waitingEventID == "" {
		if wid, ok := rawBody["waitingEventId"].(string); ok {
			waitingEventID = wid
		} else if wid, ok := rawBody["waiting_event_id"].(string); ok {
			waitingEventID = wid
		}
	}
	if waitingEventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "waiting_event_id is required"})
		return
	}

	payload := make(map[string]interface{})
	if nestedPayload, ok := rawBody["payload"].(map[string]interface{}); ok {
		payload = nestedPayload
	} else {
		for k, v := range rawBody {
			if k != "waitingEventId" && k != "waiting_event_id" {
				payload[k] = v
			}
		}
	}

	exec, err := ctrl.WorkflowExecutionBO.CompleteWaitEvent(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		waitingEventID,
		payload,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, exec)
}

func (ctrl *WorkflowExecutionController) GetGlobalWaitingEventsHandler(c *gin.Context) {
	executionID := c.Param("id")
	events, err := ctrl.WorkflowExecutionBO.GetWaitingEvents(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		executionID,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
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

func (ctrl *WorkflowExecutionController) SubmitTenantEventInputHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	var rawBody map[string]interface{}
	if err := c.ShouldBindJSON(&rawBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	waitingEventID := c.Param("waitingEventId")
	if waitingEventID == "" {
		waitingEventID = c.Param("id")
	}
	if waitingEventID == "" {
		if wid, ok := rawBody["waitingEventId"].(string); ok {
			waitingEventID = wid
		} else if wid, ok := rawBody["waiting_event_id"].(string); ok {
			waitingEventID = wid
		}
	}
	if waitingEventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "waiting_event_id is required"})
		return
	}

	payload := make(map[string]interface{})
	if nestedPayload, ok := rawBody["payload"].(map[string]interface{}); ok {
		payload = nestedPayload
	} else {
		for k, v := range rawBody {
			if k != "waitingEventId" && k != "waiting_event_id" {
				payload[k] = v
			}
		}
	}

	exec, err := ctrl.WorkflowExecutionBO.CompleteWaitEvent(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		waitingEventID,
		payload,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, exec)
}

func (ctrl *WorkflowExecutionController) GetTenantWaitingEventsHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	executionID := c.Param("id")

	events, err := ctrl.WorkflowExecutionBO.GetWaitingEvents(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		executionID,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

