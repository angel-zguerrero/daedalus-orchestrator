package tenant

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"

	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"net/http"
	"strconv"
	"strings"
	"time"

	"deadalus-orch/server/internal/infrastructure/dragonboat"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db4 "github.com/lni/dragonboat/v4"
)

type TenantController struct {
	Config *common.RestServerConfing
}

// NewTenantController creates a new instance of RestAdminAPI.
func NewTenantController(Config *common.RestServerConfing) *TenantController {

	api := &TenantController{
		Config: Config,
	}

	return api
}

func (c *TenantController) SetTenantNode(shardID int, tenantId string) *dragonboat.RaftNode {
	var tenant *dragonboat.RaftNode

	c.Config.TenantNodesLock.Lock()
	for i := range c.Config.TenantNodes {
		if c.Config.TenantNodes[i].ShardID == uint64(shardID) {
			tenant = c.Config.TenantNodes[i]
			break
		}
	}
	c.Config.TenantNodesLock.Unlock()

	c.Config.TenantNodesLock.Lock()
	c.Config.TenantNodesDictionary[tenantId] = tenant
	c.Config.TenantNodesLock.Unlock()
	return tenant
}

type createTenantInMasterRequest struct {
	Code string `json:"code" binding:"required"`
	Name string `json:"name" binding:"required"`
}

// CreateTenantHandler handles POST /admin-api/tenants
func (ctrl *TenantController) CreateTenantHandler(c *gin.Context) {
	var req createTenantInMasterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.Config.Logger.Warn().Err(err).Msg("craete tenant attempt with invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	createTenantInMasterCommand := &commands.CreateTenantInMasterCommand{
		TenantId:   strings.ReplaceAll(uuid.New().String(), "-", ""),
		TenantCode: req.Code,
		TenantName: req.Name,
	}

	fsmCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  createTenantInMasterCommand,
	}

	writeCtx, writeCancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout) // Or a specific timeout for writes
	defer writeCancel()

	result, err := ctrl.Config.MasterNode.Write(writeCtx, fsmCmd)
	if err != nil {

		ctrl.Config.Logger.Error().Err(err).Str("Code", req.Code).Msg("Failed to create new tenant")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new tenant: " + err.Error()})
		return
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("Code", req.Code).Msg("Tenant creation command returned unexpected result type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant creation command returned unexpected error"})
		return
	}

	possibleErr := parsedResult.Error
	if possibleErr != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new tenant error: " + possibleErr})
		return
	}

	tenantInMaster := parsedResult.Result.(models.TenantInMaster)

	tenantNode := ctrl.SetTenantNode(tenantInMaster.ShardId, tenantInMaster.ID)

	if tenantNode == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant node not found"})
		return
	}

	createColumnFamilyCommand := &commands.CreateColumnFamilyCommand{
		Name: tenantInMaster.ID,
	}

	ccfCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  createColumnFamilyCommand,
	}

	result, err = tenantNode.Write(writeCtx, ccfCmd)
	if err != nil {

		ctrl.Config.Logger.Error().Err(err).Str("Code", req.Code).Msg("Failed to create new tenant")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new tenant: " + err.Error()})
		return
	}

	buf = bytes.NewBuffer(result.Data)
	dec = gob.NewDecoder(buf)

	if err := dec.Decode(parsedResult); err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("Code", req.Code).Msg("Tenant creation command returned unexpected result type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant creation command returned unexpected error"})
		return
	}

	possibleErr = parsedResult.Error
	if possibleErr != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new tenant error: " + possibleErr})
		return
	}

	assignToShardTenantInMasterCommand := &commands.AssignToShardTenantInMasterCommand{
		TenantCode: req.Code,
	}

	atstCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  assignToShardTenantInMasterCommand,
	}

	result, err = ctrl.Config.MasterNode.Write(writeCtx, atstCmd)
	if err != nil {

		ctrl.Config.Logger.Error().Err(err).Str("Code", req.Code).Msg("Failed to create new tenant")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new tenant: " + err.Error()})
		return
	}

	buf = bytes.NewBuffer(result.Data)
	dec = gob.NewDecoder(buf)

	if err := dec.Decode(parsedResult); err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("Code", req.Code).Msg("Tenant creation command returned unexpected result type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant creation command returned unexpected error"})
		return
	}

	possibleErr = parsedResult.Error
	if possibleErr != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new tenant error: " + possibleErr})
		return
	}

	if parsedResult.Result.(bool) {
		tenantInMaster.Status = models.Assigned
	}

	ctrl.Config.Logger.Info().Str("code", req.Code).Msg("tenant asserted successfully")
	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant was created",
		"result":  tenantInMaster,
	})

}

// GetTenantHandler handles GET /admin-api/tenants/:id
func (ctrl *TenantController) GetTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")
	findTenantCommand := &commands.FindTenantCommand{
		TenantID: tenantID,
	}

	queryCommand := &commands.Query_Command{
		Command: &commands.Repository_Command{
			CMD: findTenantCommand,
		},
		Now: time.Now().UnixNano(), // Or handle as per specific requirements if Query_Command.Now is actively used
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := ctrl.Config.MasterNode.Read(ctx, *queryCommand)
	if err != nil {
		if strings.Contains(err.Error(), "cannot encode nil pointer of type") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
			return
		}

		ctrl.Config.Logger.Error().Err(err).Msg("Find tenants command failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Find tenants command failed: " + err.Error()})
		return
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		ctrl.Config.Logger.Error().Err(err).Msg("Find tenants command failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Find tenants command failed"})
		return
	}

	if parsedResult.Error != "" {
		ctrl.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find tenants command failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Find tenants command failed"})
		return
	}

	if parsedResult.Result == nil {
		ctrl.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find tenants command failed")
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
		return
	}

	tenantInMaster := parsedResult.Result.(models.TenantInMaster)

	node := ctrl.Config.TenantNodesDictionary[tenantInMaster.ID]

	if node == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Tenant",
			"result":  parsedResult.Result,
		})
	} else {
		nodeHostInfoOption := &db4.NodeHostInfoOption{SkipLogInfo: true}
		NodeHostInfo := node.NH.GetNodeHostInfo(*nodeHostInfoOption)
		c.JSON(http.StatusOK, gin.H{
			"message": "Tenant",
			"result":  parsedResult.Result,
			"node": gin.H{
				"SelfMember":   node.SelfMember,
				"ShardID":      node.ShardID,
				"Roles":        node.Roles,
				"NodeHostInfo": NodeHostInfo,
			},
		})
	}

}

func (ctrl *TenantController) DeleteTenantHandler(c *gin.Context) {
	tenantID := c.Param("id")

	writeCtx, writeCancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout) // Or a specific timeout for writes
	defer writeCancel()
	markToDeletionTenantInMasterCommand := &commands.MarkToDeletionTenantInMasterCommand{
		TenantId: tenantID,
	}

	atstCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  markToDeletionTenantInMasterCommand,
	}

	result, err := ctrl.Config.MasterNode.Write(writeCtx, atstCmd)
	if err != nil {

		ctrl.Config.Logger.Error().Err(err).Str("TenantID", tenantID).Msg("Failed to delete tenant")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tenant: " + err.Error()})
		return
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		ctrl.Config.Logger.Error().Err(err).Str("TenantID", tenantID).Msg("Tenant deletion command returned unexpected result type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant deletion command returned unexpected error"})
		return
	}

	possibleErr := parsedResult.Error
	if possibleErr != "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Failed to delete tenant error: " + possibleErr})
		return
	}

	deleteColumnFamilyCommand := &commands.DeleteColumnFamilyCommand{
		Name: tenantID,
	}

	ccfCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  deleteColumnFamilyCommand,
	}

	tenantNode := ctrl.Config.TenantNodesDictionary[tenantID]
	if tenantNode == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tenant error: Tenant node not found"})
		return
	}

	result, err = tenantNode.Write(writeCtx, ccfCmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tenant error: " + err.Error()})
		return
	}

	deleteTenantInMasterCommand := &commands.DeleteTenantInMasterCommand{
		TenantId: tenantID,
	}

	atstCmd = commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.REPOSITORY_COMMAND,
		CMD:  deleteTenantInMasterCommand,
	}

	result, err = ctrl.Config.MasterNode.Write(writeCtx, atstCmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tenant error: " + err.Error()})
		return
	}

	ctrl.Config.Logger.Info().Str("TenantID", tenantID).Msg("new tenant deleted successfully")
	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant " + tenantID + " was deleted",
		"result":  parsedResult.Result,
	})
}

func (ctrl *TenantController) GetTenantsHandler(c *gin.Context) {
	pageParam := c.Query("page")
	page, err := strconv.Atoi(pageParam)
	if err != nil || page < 2 {
		page = 50
	} else if page > 1000 {
		page = 1000
	}
	paginateTenantsCommand := &commands.PaginateTenantsCommand{
		Cursor:   c.Query("cursor"),
		PageSize: page,
	}

	queryCommand := &commands.Query_Command{
		Command: &commands.Repository_Command{
			CMD: paginateTenantsCommand,
		},
		Now: time.Now().UnixNano(), // Or handle as per specific requirements if Query_Command.Now is actively used
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := ctrl.Config.MasterNode.Read(ctx, *queryCommand)
	if err != nil {
		ctrl.Config.Logger.Error().Err(err).Msg("Paginate tenants command failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed: " + err.Error()})
		return
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		ctrl.Config.Logger.Error().Err(err).Msg("Paginate tenants command failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Paginate tenants command failed"})
		return
	}

	if parsedResult.Error != "" {
		ctrl.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Paginate tenants command failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Paginate tenants command failed"})
		return
	}

	findResult := parsedResult.Result.(db.FindResult[models.TenantInMaster])

	if findResult.Entities == nil {
		findResult.Entities = []models.TenantInMaster{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant list",
		"result":  findResult,
	})
}
