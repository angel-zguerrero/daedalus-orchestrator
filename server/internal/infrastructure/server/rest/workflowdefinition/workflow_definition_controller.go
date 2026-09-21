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
	OnVersionChange    models.VersionChangePolicy   `json:"onVersionChange"`
	Payload            json.RawMessage              `json:"payload"`
	PayloadFormat      models.WorkflowPayloadFormat `json:"payloadFormat"`
	MaxDurationSeconds int32                        `json:"maxDurationSeconds"`
	IsActive           *bool                        `json:"isActive"`
	VNamespace         string                       `json:"vnamespace"`
}

type updateWorkflowRequest struct {
	Name               string                       `json:"name"`
	Description        string                       `json:"description"`
	OnVersionChange    models.VersionChangePolicy   `json:"onVersionChange"`
	Payload            json.RawMessage              `json:"payload"`
	PayloadFormat      models.WorkflowPayloadFormat `json:"payloadFormat"`
	MaxDurationSeconds int32                        `json:"maxDurationSeconds"`
	IsActive           *bool                        `json:"isActive"`
	VNamespace         string                       `json:"vnamespace"`
}

func parsePayloadBytes(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return nil
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return []byte(str)
	}
	return []byte(raw)
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
		req.OnVersionChange,
		parsePayloadBytes(req.Payload),
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
	vnamespace := c.Query("vnamespace")

	res, err := ctrl.WorkflowDefinitionBO.ListWorkflows(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		"",
		vnamespace,
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
		req.OnVersionChange,
		parsePayloadBytes(req.Payload),
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

func (ctrl *WorkflowDefinitionController) ListGlobalWorkflowVersionsHandler(c *gin.Context) {
	id := c.Param("id")
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	cursor := c.Query("cursor")

	res, err := ctrl.WorkflowDefinitionBO.ListWorkflowVersions(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		id,
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

func (ctrl *WorkflowDefinitionController) GetGlobalWorkflowVersionHandler(c *gin.Context) {
	id := c.Param("id")
	verNum, _ := strconv.Atoi(c.Param("version"))
	ver, err := ctrl.WorkflowDefinitionBO.GetWorkflowVersion(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		id,
		int32(verNum),
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ver == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow version not found"})
		return
	}
	c.JSON(http.StatusOK, ver)
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
		req.OnVersionChange,
		parsePayloadBytes(req.Payload),
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
	vnamespace := c.Query("vnamespace")

	res, err := ctrl.WorkflowDefinitionBO.ListWorkflows(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		tenant.ID,
		vnamespace,
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
		req.OnVersionChange,
		parsePayloadBytes(req.Payload),
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

func (ctrl *WorkflowDefinitionController) ListTenantWorkflowVersionsHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	id := c.Param("id")
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	cursor := c.Query("cursor")

	res, err := ctrl.WorkflowDefinitionBO.ListWorkflowVersions(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		id,
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

func (ctrl *WorkflowDefinitionController) GetTenantWorkflowVersionHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	id := c.Param("id")
	verNum, _ := strconv.Atoi(c.Param("version"))

	ver, err := ctrl.WorkflowDefinitionBO.GetWorkflowVersion(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		id,
		int32(verNum),
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if ver == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow version not found"})
		return
	}
	c.JSON(http.StatusOK, ver)
}

func (ctrl *WorkflowDefinitionController) GetGlobalWorkflowQueuesHandler(c *gin.Context) {
	id := c.Param("id")
	queues, err := ctrl.WorkflowDefinitionBO.GetWorkflowQueues(
		c.Request.Context(),
		models.WorkflowScopeGlobal,
		id,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"queues": queues})
}

func (ctrl *WorkflowDefinitionController) GetTenantWorkflowQueuesHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	id := c.Param("id")
	queues, err := ctrl.WorkflowDefinitionBO.GetWorkflowQueues(
		c.Request.Context(),
		models.WorkflowScopeTenant,
		id,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"queues": queues})
}
