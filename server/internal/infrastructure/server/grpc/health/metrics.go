package health

import (
	"context"
	"os"
	"time"

	pb "deadalus-orch/server/internal/infrastructure/common/proto/health/metrics"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type MetricsServer struct {
	pb.UnimplementedMetricsServiceServer
	startTime time.Time
	NodeType  string // "main" o "follower"
}

func NewMetricsServer(nodeType string) *MetricsServer {
	return &MetricsServer{
		startTime: time.Now(),
		NodeType:  nodeType,
	}
}

func (s *MetricsServer) GetSystemMetrics(ctx context.Context, _ *pb.SystemMetricsRequest) (*pb.SystemMetricsResponse, error) {
	vmStat, _ := mem.VirtualMemory()
	cpuPercent, _ := cpu.Percent(0, false)
	hostname, _ := os.Hostname()

	return &pb.SystemMetricsResponse{
		CpuUsagePercent:  float32(cpuPercent[0]),
		MemoryTotalBytes: vmStat.Total,
		MemoryUsedBytes:  vmStat.Used,
		MemoryFreeBytes:  vmStat.Free,
		UptimeSeconds:    uint64(time.Since(s.startTime).Seconds()),
		Hostname:         hostname,
		NodeType:         s.NodeType,
		Timestamp:        time.Now().Unix(),
	}, nil
}
