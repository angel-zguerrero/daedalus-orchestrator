package activitytemplate

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"

	"github.com/gin-gonic/gin"
)

type ActivityTemplateController struct {
	Config             *common.ServerConfing
	ActivityTemplateBO *bo.ActivityTemplateBO
	TenantBO           *bo.TenantBO
}

func NewActivityTemplateController(config *common.ServerConfing) *ActivityTemplateController {
	return &ActivityTemplateController{
		Config:             config,
		ActivityTemplateBO: bo.NewActivityTemplateBO(config),
		TenantBO:           bo.NewTenantBO(config),
	}
}

type createActivityTemplateRequest struct {
	Code             string          `json:"code"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	ActivityFamily   string          `json:"activityFamily"`
	ParentTemplateId string          `json:"parentTemplateId"`
	RootActivity     string          `json:"rootActivity"`
	Payload          json.RawMessage `json:"payload"`
	IsActive         *bool           `json:"isActive"`
	VNamespace       string          `json:"vnamespace"`
}

type updateActivityTemplateRequest struct {
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	ActivityFamily   string          `json:"activityFamily"`
	ParentTemplateId string          `json:"parentTemplateId"`
	RootActivity     string          `json:"rootActivity"`
	Payload          json.RawMessage `json:"payload"`
	IsActive         *bool           `json:"isActive"`
	VNamespace       string          `json:"vnamespace"`
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

func (ctrl *ActivityTemplateController) CreateGlobalActivityTemplateHandler(c *gin.Context) {
	var req createActivityTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	tpl, err := ctrl.ActivityTemplateBO.CreateActivityTemplate(
		c.Request.Context(),
		models.ActivityTemplateScopeGlobal,
		"",
		req.Code,
		req.Name,
		req.Description,
		req.ActivityFamily,
		req.ParentTemplateId,
		req.RootActivity,
		parsePayloadBytes(req.Payload),
		isActive,
		req.VNamespace,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tpl)
}

func (ctrl *ActivityTemplateController) ListGlobalActivityTemplatesHandler(c *gin.Context) {
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	cursor := c.Query("cursor")
	vnamespace := c.Query("vnamespace")
	activityFamily := c.Query("activityFamily")

	res, err := ctrl.ActivityTemplateBO.ListActivityTemplates(
		c.Request.Context(),
		models.ActivityTemplateScopeGlobal,
		"",
		vnamespace,
		activityFamily,
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

func (ctrl *ActivityTemplateController) GetGlobalActivityTemplateHandler(c *gin.Context) {
	id := c.Param("id")
	tpl, err := ctrl.ActivityTemplateBO.GetActivityTemplate(
		c.Request.Context(),
		models.ActivityTemplateScopeGlobal,
		id,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tpl == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity template not found"})
		return
	}

	c.JSON(http.StatusOK, tpl)
}

func (ctrl *ActivityTemplateController) UpdateGlobalActivityTemplateHandler(c *gin.Context) {
	id := c.Param("id")
	var req updateActivityTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	tpl, err := ctrl.ActivityTemplateBO.UpdateActivityTemplate(
		c.Request.Context(),
		models.ActivityTemplateScopeGlobal,
		id,
		req.Name,
		req.Description,
		req.ActivityFamily,
		req.ParentTemplateId,
		req.RootActivity,
		parsePayloadBytes(req.Payload),
		isActive,
		req.VNamespace,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tpl)
}

func (ctrl *ActivityTemplateController) DeleteGlobalActivityTemplateHandler(c *gin.Context) {
	id := c.Param("id")
	err := ctrl.ActivityTemplateBO.DeleteActivityTemplate(
		c.Request.Context(),
		models.ActivityTemplateScopeGlobal,
		id,
		"", "", nil,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activity template deleted successfully"})
}

// ListForDesignerHandler returns custom ActivityTemplates for the BPMN Designer lazy-fetch.
// If tenantCode is empty, it returns only Global templates.
// If tenantCode is provided, it returns Global templates first, followed by Tenant templates for that tenant.
func (ctrl *ActivityTemplateController) ListForDesignerHandler(c *gin.Context) {
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize <= 0 {
		pageSize = 20
	}
	cursor := c.Query("cursor")
	tenantCode := strings.TrimSpace(c.Query("tenantCode"))
	if tenantCode == "" {
		tenantCode = strings.TrimSpace(c.Param("code"))
	}

	var combined []models.ActivityTemplate
	var nextCursor string

	// Cursor format:
	// "" or "g:<cursor>" -> fetching global templates first
	// "t:<cursor>"       -> fetching tenant templates
	phase := "g"
	innerCursor := ""
	if strings.HasPrefix(cursor, "t:") {
		phase = "t"
		innerCursor = strings.TrimPrefix(cursor, "t:")
	} else if strings.HasPrefix(cursor, "g:") {
		phase = "g"
		innerCursor = strings.TrimPrefix(cursor, "g:")
	} else {
		innerCursor = cursor
	}

	if phase == "g" {
		globalRes, err := ctrl.ActivityTemplateBO.ListActivityTemplates(
			c.Request.Context(),
			models.ActivityTemplateScopeGlobal,
			"",
			"",
			"",
			pageSize,
			innerCursor,
			"", "", nil,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if globalRes != nil {
			for _, item := range globalRes.Entities {
				if item.IsActive {
					combined = append(combined, item)
				}
			}
			if globalRes.Cursor != "" && len(globalRes.Entities) >= pageSize {
				nextCursor = "g:" + globalRes.Cursor
			} else if tenantCode != "" {
				// Global exhausted; if we still have room in pageSize, fetch tenant templates now
				remaining := pageSize - len(combined)
				if remaining > 0 {
					tEntities, tCur := ctrl.fetchTenantTemplatesByCode(c, tenantCode, remaining, "")
					combined = append(combined, tEntities...)
					if tCur != "" {
						nextCursor = "t:" + tCur
					}
				} else {
					nextCursor = "t:"
				}
			}
		}
	} else if phase == "t" && tenantCode != "" {
		tEntities, tCur := ctrl.fetchTenantTemplatesByCode(c, tenantCode, pageSize, innerCursor)
		combined = append(combined, tEntities...)
		if tCur != "" {
			nextCursor = "t:" + tCur
		}
	}

	c.JSON(http.StatusOK, db.FindResult[models.ActivityTemplate]{
		Entities: combined,
		Cursor:   nextCursor,
	})
}

func (ctrl *ActivityTemplateController) fetchTenantTemplatesByCode(c *gin.Context, tenantCode string, pageSize int, cursor string) ([]models.ActivityTemplate, string) {
	tenant, _, _, err := ctrl.TenantBO.GetTenant(c.Request.Context(), tenantCode)
	if err != nil || tenant.ID == "" {
		return nil, ""
	}

	cf := db.ColumnFamilyPrefix + strconv.Itoa(tenant.ColumnFamilyIndex)
	cfs := tenant.ID

	var node *dragonboat.RaftNode
	ctrl.Config.TenantNodesLock.Lock()
	for i := range ctrl.Config.TenantNodes {
		if ctrl.Config.TenantNodes[i].ShardID == uint64(tenant.ShardId) {
			node = ctrl.Config.TenantNodes[i]
			break
		}
	}
	ctrl.Config.TenantNodesLock.Unlock()
	if node == nil {
		return nil, ""
	}

	tRes, err := ctrl.ActivityTemplateBO.ListActivityTemplates(
		c.Request.Context(),
		models.ActivityTemplateScopeTenant,
		tenant.ID,
		"",
		"",
		pageSize,
		cursor,
		cf, cfs, node,
	)
	if err != nil || tRes == nil {
		return nil, ""
	}

	var active []models.ActivityTemplate
	for _, item := range tRes.Entities {
		if item.IsActive {
			active = append(active, item)
		}
	}
	nextCur := ""
	if tRes.Cursor != "" && len(tRes.Entities) >= pageSize {
		nextCur = tRes.Cursor
	}
	return active, nextCur
}

// --- TENANT HANDLERS ---

func (ctrl *ActivityTemplateController) CreateTenantActivityTemplateHandler(c *gin.Context) {
	var req createActivityTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	tpl, err := ctrl.ActivityTemplateBO.CreateActivityTemplate(
		c.Request.Context(),
		models.ActivityTemplateScopeTenant,
		tenant.ID,
		req.Code,
		req.Name,
		req.Description,
		req.ActivityFamily,
		req.ParentTemplateId,
		req.RootActivity,
		parsePayloadBytes(req.Payload),
		isActive,
		req.VNamespace,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tpl)
}

func (ctrl *ActivityTemplateController) ListTenantActivityTemplatesHandler(c *gin.Context) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	cursor := c.Query("cursor")
	vnamespace := c.Query("vnamespace")
	activityFamily := c.Query("activityFamily")

	res, err := ctrl.ActivityTemplateBO.ListActivityTemplates(
		c.Request.Context(),
		models.ActivityTemplateScopeTenant,
		tenant.ID,
		vnamespace,
		activityFamily,
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

func (ctrl *ActivityTemplateController) GetTenantActivityTemplateHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	id := c.Param("id")

	tpl, err := ctrl.ActivityTemplateBO.GetActivityTemplate(
		c.Request.Context(),
		models.ActivityTemplateScopeTenant,
		id,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tpl == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Activity template not found"})
		return
	}

	c.JSON(http.StatusOK, tpl)
}

func (ctrl *ActivityTemplateController) UpdateTenantActivityTemplateHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	id := c.Param("id")
	var req updateActivityTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	tpl, err := ctrl.ActivityTemplateBO.UpdateActivityTemplate(
		c.Request.Context(),
		models.ActivityTemplateScopeTenant,
		id,
		req.Name,
		req.Description,
		req.ActivityFamily,
		req.ParentTemplateId,
		req.RootActivity,
		parsePayloadBytes(req.Payload),
		isActive,
		req.VNamespace,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tpl)
}

func (ctrl *ActivityTemplateController) DeleteTenantActivityTemplateHandler(c *gin.Context) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(c.Request.Context())
	id := c.Param("id")

	err := ctrl.ActivityTemplateBO.DeleteActivityTemplate(
		c.Request.Context(),
		models.ActivityTemplateScopeTenant,
		id,
		cf, cfs, tenantNode,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activity template deleted successfully"})
}
