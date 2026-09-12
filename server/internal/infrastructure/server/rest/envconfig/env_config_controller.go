package envconfig

import (
	"net/http"
	"strconv"

	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"

	"github.com/gin-gonic/gin"
)

type EnvConfigController struct {
	Config     *common.ServerConfing
	EnvGroupBO *bo.EnvGroupBO
}

func NewEnvConfigController(config *common.ServerConfing) *EnvConfigController {
	return &EnvConfigController{
		Config:     config,
		EnvGroupBO: bo.NewEnvGroupBO(config),
	}
}

type createEnvGroupRequest struct {
	Code        string              `json:"code"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Type        models.EnvGroupType `json:"type"`
	VNamespace  string              `json:"vnamespace"`
}

type updateEnvGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type saveEnvVarRequest struct {
	ID          string `json:"id"`
	Key         string `json:"key" binding:"required"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type bulkSaveEnvVarsRequest struct {
	Vars []models.EnvVar `json:"vars" binding:"required"`
}

// --- GLOBAL HANDLERS ---

func (ctrl *EnvConfigController) CreateGlobalGroupHandler(c *gin.Context) {
	var req createEnvGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	group, err := ctrl.EnvGroupBO.CreateGroup(
		c.Request.Context(),
		models.EnvGroupScopeGlobal,
		"",
		req.Code,
		req.Name,
		req.Description,
		req.Type,
		req.VNamespace,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

func (ctrl *EnvConfigController) ListGlobalGroupsHandler(c *gin.Context) {
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	cursor := c.Query("cursor")

	res, err := ctrl.EnvGroupBO.ListGroups(
		c.Request.Context(),
		models.EnvGroupScopeGlobal,
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

func (ctrl *EnvConfigController) GetGlobalGroupHandler(c *gin.Context) {
	groupID := c.Param("groupId")
	group, err := ctrl.EnvGroupBO.GetGroup(
		c.Request.Context(),
		models.EnvGroupScopeGlobal,
		groupID,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if group == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Env group not found"})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (ctrl *EnvConfigController) UpdateGlobalGroupHandler(c *gin.Context) {
	groupID := c.Param("groupId")
	var req updateEnvGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	group, err := ctrl.EnvGroupBO.UpdateGroup(
		c.Request.Context(),
		models.EnvGroupScopeGlobal,
		groupID,
		req.Name,
		req.Description,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (ctrl *EnvConfigController) DeleteGlobalGroupHandler(c *gin.Context) {
	groupID := c.Param("groupId")
	err := ctrl.EnvGroupBO.DeleteGroup(
		c.Request.Context(),
		models.EnvGroupScopeGlobal,
		groupID,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Env group deleted successfully"})
}

func (ctrl *EnvConfigController) GetGlobalVarsHandler(c *gin.Context) {
	groupID := c.Param("groupId")
	vars, err := ctrl.EnvGroupBO.GetGroupVars(
		c.Request.Context(),
		models.EnvGroupScopeGlobal,
		groupID,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entities": vars})
}

func (ctrl *EnvConfigController) SaveGlobalVarHandler(c *gin.Context) {
	groupID := c.Param("groupId")
	var req saveEnvVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	saved, err := ctrl.EnvGroupBO.SaveGroupVar(
		c.Request.Context(),
		models.EnvGroupScopeGlobal,
		groupID,
		req.ID,
		req.Key,
		req.Value,
		req.Description,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, saved)
}

func (ctrl *EnvConfigController) DeleteGlobalVarHandler(c *gin.Context) {
	varID := c.Param("varId")
	err := ctrl.EnvGroupBO.DeleteGroupVar(
		c.Request.Context(),
		models.EnvGroupScopeGlobal,
		varID,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Env variable deleted successfully"})
}

func (ctrl *EnvConfigController) BulkSaveGlobalVarsHandler(c *gin.Context) {
	groupID := c.Param("groupId")
	var req bulkSaveEnvVarsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	saved, err := ctrl.EnvGroupBO.BulkSaveGroupVars(
		c.Request.Context(),
		models.EnvGroupScopeGlobal,
		groupID,
		req.Vars,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entities": saved})
}

// --- TENANT HANDLERS ---

func (ctrl *EnvConfigController) CreateTenantGroupHandler(c *gin.Context) {
	var req createEnvGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	group, err := ctrl.EnvGroupBO.CreateGroup(
		c.Request.Context(),
		models.EnvGroupScopeTenant,
		tenant.ID,
		req.Code,
		req.Name,
		req.Description,
		req.Type,
		req.VNamespace,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

func (ctrl *EnvConfigController) ListTenantGroupsHandler(c *gin.Context) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	cursor := c.Query("cursor")

	res, err := ctrl.EnvGroupBO.ListGroups(
		c.Request.Context(),
		models.EnvGroupScopeTenant,
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

func (ctrl *EnvConfigController) GetTenantGroupHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	groupID := c.Param("groupId")

	group, err := ctrl.EnvGroupBO.GetGroup(
		c.Request.Context(),
		models.EnvGroupScopeTenant,
		groupID,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if group == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Env group not found"})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (ctrl *EnvConfigController) UpdateTenantGroupHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	groupID := c.Param("groupId")
	var req updateEnvGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	group, err := ctrl.EnvGroupBO.UpdateGroup(
		c.Request.Context(),
		models.EnvGroupScopeTenant,
		groupID,
		req.Name,
		req.Description,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (ctrl *EnvConfigController) DeleteTenantGroupHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	groupID := c.Param("groupId")

	err := ctrl.EnvGroupBO.DeleteGroup(
		c.Request.Context(),
		models.EnvGroupScopeTenant,
		groupID,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Env group deleted successfully"})
}

func (ctrl *EnvConfigController) GetTenantVarsHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	groupID := c.Param("groupId")

	vars, err := ctrl.EnvGroupBO.GetGroupVars(
		c.Request.Context(),
		models.EnvGroupScopeTenant,
		groupID,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entities": vars})
}

func (ctrl *EnvConfigController) SaveTenantVarHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	groupID := c.Param("groupId")
	var req saveEnvVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	saved, err := ctrl.EnvGroupBO.SaveGroupVar(
		c.Request.Context(),
		models.EnvGroupScopeTenant,
		groupID,
		req.ID,
		req.Key,
		req.Value,
		req.Description,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, saved)
}

func (ctrl *EnvConfigController) DeleteTenantVarHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	varID := c.Param("varId")

	err := ctrl.EnvGroupBO.DeleteGroupVar(
		c.Request.Context(),
		models.EnvGroupScopeTenant,
		varID,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Env variable deleted successfully"})
}

func (ctrl *EnvConfigController) BulkSaveTenantVarsHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	groupID := c.Param("groupId")
	var req bulkSaveEnvVarsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	saved, err := ctrl.EnvGroupBO.BulkSaveGroupVars(
		c.Request.Context(),
		models.EnvGroupScopeTenant,
		groupID,
		req.Vars,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entities": saved})
}
