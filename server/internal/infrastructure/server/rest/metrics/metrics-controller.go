package metrics

import (
	"deadalus-orch/server/internal/infrastructure/server/rest/common"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// MetricsController handles the administrative REST API endpoints.
type MetricsController struct {
	Config *common.RestServerConfing
}

// NewMetricsController creates a new instance of RestAdminAPI.
func NewMetricsController(Config *common.RestServerConfing) *MetricsController {

	api := &MetricsController{
		Config: Config,
	}

	return api
}

// GetTenantHandler handles GET /metrics
func (ctrl *MetricsController) GetSystemMetricsHandler(c *gin.Context) {

	vmStat, _ := mem.VirtualMemory()
	cpuPercent, _ := cpu.Percent(0, false)
	hostname, _ := os.Hostname()

	c.JSON(http.StatusOK, gin.H{
		"CpuUsagePercent":  float32(cpuPercent[0]),
		"MemoryTotalBytes": vmStat.Total,
		"MemoryUsedBytes":  vmStat.Used,
		"MemoryFreeBytes":  vmStat.Free,
		"Hostname":         hostname,
		"NodeRoles":        ctrl.Config.MasterNode.Roles,
		"Timestamp":        time.Now().Unix(),
	})

}
