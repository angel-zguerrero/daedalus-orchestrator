package workflowdefinition

import (
	"encoding/json"
	"net/http"
	"strconv"

	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"

	"github.com/gin-gonic/gin"
)

type WorkflowDefinitionController struct {
	Config               *common.ServerConfing
	WorkflowDefinitionBO *bo.WorkflowDefinitionBO
}

func NewWorkflowDefinitionController(config *common.ServerConfing) *WorkflowDefinitionController {
	return &WorkflowDefinitionController{
		Config:               config,
		WorkflowDefinitionBO: bo.NewWorkflowDefinitionBO(config),
	}
}

type createWorkflowRequest struct {
	Code               string                       `json:"code"`
	Name               string                       `json:"name"`
	Description        string                       `json:"description"`
	Version            int32                        `json:"version"`
	Payload            json.RawMessage              `json:"payload"`
	PayloadFormat      models.WorkflowPayloadFormat `json:"payloadFormat"`
	MaxDurationSeconds int32                        `json:"maxDurationSeconds"`
	IsActive           *bool                        `json:"isActive"`
	VNamespace         string                       `json:"vnamespace"`
}

type updateWorkflowRequest struct {
	Name               string                       `json:"name"`
	Description        string                       `json:"description"`
	Version            int32                        `json:"version"`
	Payload            json.RawMessage              `json:"payload"`
	PayloadFormat      models.WorkflowPayloadFormat `json:"payloadFormat"`
	MaxDurationSeconds int32                        `json:"maxDurationSeconds"`
	IsActive           *bool                        `json:"isActive"`
}

// --- GLOBAL HANDLERS ---

func (ctrl *WorkflowDefinitionController) CreateGlobalWorkflowHandler(c *gin.Context) {
	var req createWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	wf, err := ctrl.WorkflowDefinitionBO.CreateWorkflow(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		"",
		req.Code,
		req.Name,
		req.Description,
		req.Version,
		[]byte(req.Payload),
		req.PayloadFormat,
		req.MaxDurationSeconds,
		isActive,
		req.VNamespace,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, wf)
}

func (ctrl *WorkflowDefinitionController) ListGlobalWorkflowsHandler(c *gin.Context) {
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	cursor := c.Query("cursor")

	res, err := ctrl.WorkflowDefinitionBO.ListWorkflows(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		"",
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

func (ctrl *WorkflowDefinitionController) GetGlobalWorkflowHandler(c *gin.Context) {
	id := c.Param("id")
	wf, err := ctrl.WorkflowDefinitionBO.GetWorkflow(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		id,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if wf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow definition not found"})
		return
	}

	c.JSON(http.StatusOK, wf)
}

func (ctrl *WorkflowDefinitionController) UpdateGlobalWorkflowHandler(c *gin.Context) {
	id := c.Param("id")
	var req updateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	wf, err := ctrl.WorkflowDefinitionBO.UpdateWorkflow(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		id,
		req.Name,
		req.Description,
		req.Version,
		[]byte(req.Payload),
		req.PayloadFormat,
		req.MaxDurationSeconds,
		isActive,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wf)
}

func (ctrl *WorkflowDefinitionController) DeleteGlobalWorkflowHandler(c *gin.Context) {
	id := c.Param("id")
	err := ctrl.WorkflowDefinitionBO.DeleteWorkflow(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		id,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Workflow definition deleted successfully"})
}

// --- TENANT HANDLERS ---

func (ctrl *WorkflowDefinitionController) CreateTenantWorkflowHandler(c *gin.Context) {
	var req createWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	wf, err := ctrl.WorkflowDefinitionBO.CreateWorkflow(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		tenant.ID,
		req.Code,
		req.Name,
		req.Description,
		req.Version,
		[]byte(req.Payload),
		req.PayloadFormat,
		req.MaxDurationSeconds,
		isActive,
		req.VNamespace,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, wf)
}

func (ctrl *WorkflowDefinitionController) ListTenantWorkflowsHandler(c *gin.Context) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	cursor := c.Query("cursor")

	res, err := ctrl.WorkflowDefinitionBO.ListWorkflows(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		tenant.ID,
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

func (ctrl *WorkflowDefinitionController) GetTenantWorkflowHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	id := c.Param("id")

	wf, err := ctrl.WorkflowDefinitionBO.GetWorkflow(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		id,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if wf == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow definition not found"})
		return
	}

	c.JSON(http.StatusOK, wf)
}

func (ctrl *WorkflowDefinitionController) UpdateTenantWorkflowHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	id := c.Param("id")
	var req updateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	wf, err := ctrl.WorkflowDefinitionBO.UpdateWorkflow(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		id,
		req.Name,
		req.Description,
		req.Version,
		[]byte(req.Payload),
		req.PayloadFormat,
		req.MaxDurationSeconds,
		isActive,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wf)
}

func (ctrl *WorkflowDefinitionController) DeleteTenantWorkflowHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	id := c.Param("id")

	err := ctrl.WorkflowDefinitionBO.DeleteWorkflow(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		id,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Workflow definition deleted successfully"})
}
