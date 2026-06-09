package cluster

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"


)

// ClusterController handles cluster management operations like adding/removing nodes
type ClusterController struct {
	Config *common.ServerConfing
}

// NewClusterController creates a new instance of ClusterController
func NewClusterController(Config *common.ServerConfing) *ClusterController {
	return &ClusterController{
		Config: Config,
	}
}

// AddReplicaRequest represents the request body for adding a replica to the cluster
type AddReplicaRequest struct {
	ReplicaID uint64 `json:"replica_id" binding:"required" example:"4"`
	Host      string `json:"host" binding:"required" example:"127.0.0.1"`
	Port      int    `json:"port" binding:"required" example:"5004"`
}

// AddReplicaResponse represents the response for adding a replica
type AddReplicaResponse struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	ReplicaID uint64                 `json:"replica_id"`
	Results   map[uint64]ShardResult `json:"results"` // Map of ShardID -> Result
}

// ShardResult represents the result for a specific shard operation
type ShardResult struct {
	ShardID uint64 `json:"shard_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// AddReplica adds a new replica to the specified shards in the cluster
// @Summary Add a new replica to cluster shards
// @Description Adds a new replica to one or more shards in the Dragonboat cluster. The replica will be added to the master shard and all specified tenant shards.
// @Tags Cluster Management
// @Accept json
// @Produce json
// @Param request body AddReplicaRequest true "Add replica request"
// @Success 200 {object} AddReplicaResponse "Replica addition results"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/cluster/replicas [post]
func (cc *ClusterController) AddReplica(c *gin.Context) {
	var req AddReplicaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Msg("❌ Invalid request for adding replica")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	log.Info().
		Uint64("replica_id", req.ReplicaID).
		Str("host", req.Host).
		Int("port", req.Port).
		Msg("🔄 Processing add replica request")

	// Create member from request
	member := dragonboat.Member{
		IP:   req.Host,
		Port: req.Port,
	}

	// Determine which shards to add replica to

	// If no specific shards provided, add to all available shards
	shardIDs := []uint64{dragonboat.MasterShardID} // Start with master shard
	for _, tenantNode := range cc.Config.TenantNodes {
		shardIDs = append(shardIDs, tenantNode.ShardID)
	}

	results := make(map[uint64]ShardResult)
	successCount := 0

	// Process each shard
	for _, shardID := range shardIDs {
		result := ShardResult{
			ShardID: shardID,
			Success: false,
		}

		var raftNode *dragonboat.RaftNode
		var shardName string

		// Get the appropriate RaftNode for this shard
		if shardID == dragonboat.MasterShardID {
			raftNode = cc.Config.MasterNode
			shardName = "Master"
		} else {
			// Find tenant node with matching ShardID
			for _, tenantNode := range cc.Config.TenantNodes {
				if tenantNode.ShardID == shardID {
					raftNode = tenantNode
					shardName = "Tenant"
					break
				}
			}
		}

		if raftNode == nil {
			result.Error = "Shard not found on this node"
			result.Message = "Shard not available"
			log.Warn().Uint64("shard_id", shardID).Msg("⚠️ Shard not found on this node")
		} else {
			// Attempt to add replica to this shard
			err := raftNode.RequestAddReplica(req.ReplicaID, member)
			if err != nil {
				result.Error = err.Error()
				result.Message = "Failed to add replica"
				log.Error().
					Err(err).
					Uint64("shard_id", shardID).
					Str("shard_name", shardName).
					Msg("❌ Failed to add replica to shard")
			} else {
				result.Success = true
				result.Message = "Replica added successfully"
				successCount++
				log.Info().
					Uint64("shard_id", shardID).
					Str("shard_name", shardName).
					Uint64("replica_id", req.ReplicaID).
					Msg("✅ Replica added to shard successfully")
			}
		}

		results[shardID] = result
	}

	// Prepare response
	response := AddReplicaResponse{
		Success:   successCount > 0,
		ReplicaID: req.ReplicaID,
		Results:   results,
	}

	if successCount == len(shardIDs) {
		response.Message = "Replica added to all requested shards successfully"
		log.Info().
			Uint64("replica_id", req.ReplicaID).
			Int("shard_count", successCount).
			Msg("✅ Replica added to all shards successfully")
		c.JSON(http.StatusOK, response)
	} else if successCount > 0 {
		response.Message = "Replica added to some shards with partial success"
		log.Warn().
			Uint64("replica_id", req.ReplicaID).
			Int("success_count", successCount).
			Int("total_count", len(shardIDs)).
			Msg("⚠️ Replica added with partial success")
		c.JSON(http.StatusOK, response)
	} else {
		response.Message = "Failed to add replica to any shard"
		log.Error().
			Uint64("replica_id", req.ReplicaID).
			Msg("❌ Failed to add replica to any shard")
		c.JSON(http.StatusInternalServerError, response)
	}
}

// ClusterNodeInfo represents information about a node in the cluster from GetClusterConfig
type ClusterNodeInfo struct {
	NodeID  uint64 `json:"node_id"`
	Address string `json:"address"`
	ShardID uint64 `json:"shard_id"`
}

// ClusterConfigInfo represents the cluster configuration information
type ClusterConfigInfo struct {
	ShardID uint64            `json:"shard_id"`
	Nodes   []ClusterNodeInfo `json:"nodes"`
	Total   int               `json:"total"`
}

// ClusterPortInfo represents the port configuration information for the cluster
type ClusterPortInfo struct {
	ClusterBasePort int `json:"cluster_base_port"`
	RestPort        int `json:"rest_port"`
	GrpcPort        int `json:"grpc_port"`
}

// EnhancedClusterInfo represents the complete cluster information
type EnhancedClusterInfo struct {
	// Primary information from GetClusterConfig
	ClusterConfig []ClusterConfigInfo `json:"cluster_config"`

	// Secondary information (current node configuration)
	NodeConfiguration struct {
		MasterNode  gin.H   `json:"master_node"`
		TenantNodes []gin.H `json:"tenant_nodes"`
	} `json:"node_configuration"`


	// Port configuration information
	PortConfiguration ClusterPortInfo `json:"port_configuration"`
}

// GetClusterInfo gets information about the current cluster state using GetClusterConfig as primary data
// @Summary Get cluster information
// @Description Retrieves information about the current cluster state including live cluster configuration and node details
// @Tags Cluster Management
// @Produce json
// @Success 200 {object} EnhancedClusterInfo "Enhanced cluster information"
// @Router /api/v1/cluster/info [get]
func (cc *ClusterController) GetClusterInfo(c *gin.Context) {
	response := EnhancedClusterInfo{
		ClusterConfig: []ClusterConfigInfo{},
	}

	// Get cluster configuration from all available shards using GetClusterConfig
	shardsToCheck := []uint64{dragonboat.MasterShardID}
	for _, tenantNode := range cc.Config.TenantNodes {
		shardsToCheck = append(shardsToCheck, tenantNode.ShardID)
	}

	// Extract cluster config information from each shard
	for _, shardID := range shardsToCheck {
		var raftNode *dragonboat.RaftNode

		// Get the appropriate RaftNode for this shard
		if shardID == dragonboat.MasterShardID {
			raftNode = cc.Config.MasterNode
		} else {
			// Find tenant node with matching ShardID
			for _, tenantNode := range cc.Config.TenantNodes {
				if tenantNode.ShardID == shardID {
					raftNode = tenantNode
					break
				}
			}
		}

		if raftNode != nil && raftNode.NH != nil {
			// Use SyncGetShardMembership to get live cluster configuration
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			membership, err := raftNode.NH.SyncGetShardMembership(ctx, shardID)
			if err != nil {
				log.Warn().
					Uint64("shard_id", shardID).
					Err(err).
					Msg("⚠️ Failed to get shard membership for shard")
				continue
			}

			// Build cluster config info for this shard
			clusterConfigInfo := ClusterConfigInfo{
				ShardID: shardID,
				Nodes:   []ClusterNodeInfo{},
				Total:   len(membership.Nodes),
			}

			// Extract node information from cluster membership
			for nodeID, addr := range membership.Nodes {
				nodeInfo := ClusterNodeInfo{
					NodeID:  nodeID,
					Address: addr,
					ShardID: shardID,
				}
				clusterConfigInfo.Nodes = append(clusterConfigInfo.Nodes, nodeInfo)

				log.Debug().
					Uint64("shard_id", shardID).
					Uint64("node_id", nodeID).
					Str("addr", addr).
					Msg("🔍 Found cluster node")
			}

			// Also include non-voting nodes if any
			for nodeID, addr := range membership.NonVotings {
				nodeInfo := ClusterNodeInfo{
					NodeID:  nodeID,
					Address: addr + " (non-voting)",
					ShardID: shardID,
				}
				clusterConfigInfo.Nodes = append(clusterConfigInfo.Nodes, nodeInfo)
				clusterConfigInfo.Total++

				log.Debug().
					Uint64("shard_id", shardID).
					Uint64("node_id", nodeID).
					Str("addr", addr).
					Msg("🔍 Found non-voting cluster node")
			}

			// Also include witness nodes if any
			for nodeID, addr := range membership.Witnesses {
				nodeInfo := ClusterNodeInfo{
					NodeID:  nodeID,
					Address: addr + " (witness)",
					ShardID: shardID,
				}
				clusterConfigInfo.Nodes = append(clusterConfigInfo.Nodes, nodeInfo)
				clusterConfigInfo.Total++

				log.Debug().
					Uint64("shard_id", shardID).
					Uint64("node_id", nodeID).
					Str("addr", addr).
					Msg("🔍 Found witness cluster node")
			}

			response.ClusterConfig = append(response.ClusterConfig, clusterConfigInfo)
		}
	}

	// Secondary information: Current node configuration
	response.NodeConfiguration.MasterNode = gin.H{
		"replica_id":  cc.Config.MasterNode.ReplicaID,
		"shard_id":    cc.Config.MasterNode.ShardID,
		"self_member": cc.Config.MasterNode.SelfMember,
		"roles":       cc.Config.MasterNode.Roles,
	}

	response.NodeConfiguration.TenantNodes = make([]gin.H, 0)
	for tenantID, node := range cc.Config.TenantNodes {
		response.NodeConfiguration.TenantNodes = append(response.NodeConfiguration.TenantNodes, gin.H{
			"tenant_id":   tenantID,
			"replica_id":  node.ReplicaID,
			"shard_id":    node.ShardID,
			"self_member": node.SelfMember,
			"roles":       node.Roles,
		})
	}

	log.Info().
		Int("cluster_shards", len(response.ClusterConfig)).
		Msg("✅ Retrieved enhanced cluster information")



	// Add port configuration information
	response.PortConfiguration = ClusterPortInfo{
		ClusterBasePort: config.GlobalConfiguration.ClusterBasePort,
		RestPort:        config.GlobalConfiguration.RestListenAddrPort,
		GrpcPort:        config.GlobalConfiguration.GrpcServerListenAddrPort,
	}

	c.JSON(http.StatusOK, response)
}

// RegisterRoutes registers the cluster management routes with the provided router group
func (cc *ClusterController) RegisterRoutes(rg *gin.RouterGroup) {
	cluster := rg.Group("/cluster")
	{
		cluster.POST("/replicas", cc.AddReplica)
		cluster.GET("/info", cc.GetClusterInfo)
	}
}
