package rest_api_admin

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// getTenantsHandler handles GET /admin-api/tenants
func (api *RestAdminAPI) getTenantsHandler(c *gin.Context) {

	c.JSON(http.StatusOK, gin.H{"message": "Listing all tenants (dummy)", "tenants": []string{"tenantA", "tenantttB", "tenantC"}})
}

type createTenantInMasterRequest struct {
	Code string `json:"code" binding:"required"`
}

// createTenantHandler handles POST /admin-api/tenants
func (api *RestAdminAPI) createTenantHandler(c *gin.Context) {
	var req createTenantInMasterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Warn().Err(err).Msg("craete tenant attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	createTenantInMasterCommand := &commands.CreateTenantInMasterCommand{
		TenantId:   strings.ReplaceAll(uuid.New().String(), "-", ""),
		TenantCode: req.Code,
	}

	fsmCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  createTenantInMasterCommand,
	}

	writeCtx, writeCancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout) // Or a specific timeout for writes
	defer writeCancel()

	result, err := api.MasterNode.Write(writeCtx, fsmCmd)
	if err != nil {

		api.logger.Error().Err(err).Str("Code", req.Code).Msg("Failed to create new tenant")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new tenant: " + err.Error()})
		return
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		api.logger.Error().Err(err).Str("Code", req.Code).Msg("Tenant creation command returned unexpected result type!")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant creation command returned unexpected error"})
		return
	}

	possibleErr := parsedResult.Error
	if possibleErr != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new tenant error: " + possibleErr})
		return
	}

	tenantInMaster := parsedResult.Result.(models.TenantInMaster)

	tenantNode := api.SetTenantNode(tenantInMaster.ShardId, tenantInMaster.ID)

	if tenantNode == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant node not found"})
		return
	}

	api.logger.Info().Str("code", req.Code).Msg("new tenant created successfully")
	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant was created",
		"result":  tenantInMaster,
	})
}

// getTenantHandler handles GET /admin-api/tenants/:id
func (api *RestAdminAPI) getTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"message": "Details for tenant " + tenantID + " (dummy)", "id": tenantID, "name": "Dummy Tenant " + tenantID})
}

// updateTenantHandler handles PUT /admin-api/tenants/:id
func (api *RestAdminAPI) updateTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"message": "Tenant " + tenantID + " updated (dummy)", "id": tenantID})
}

// deleteTenantHandler handles DELETE /admin-api/tenants/:id
func (api *RestAdminAPI) deleteTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")

	c.JSON(http.StatusOK, gin.H{"message": "Tenant " + tenantID + " deleted (dummy)"})
}
