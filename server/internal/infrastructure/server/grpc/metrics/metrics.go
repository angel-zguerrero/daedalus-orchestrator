package health

import (
	"context"
	"os"
	"time"

	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/health/metrics"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// MetricsServer implements the pb.MetricsServiceServer interface.
// It provides gRPC methods to retrieve system metrics such as CPU usage,
// memory usage, uptime, hostname, and node type.
type MetricsServer struct {
	pb.UnimplementedMetricsServiceServer           // Embeds the unimplemented server for forward compatibility.
	startTime                            time.Time // The time when the MetricsServer was instantiated, used to calculate uptime.
	NodeType                             string    // Type of the node (e.g., "main", "follower") this metrics server is running on.
}

// NewMetricsServer creates a new instance of MetricsServer.
// It records the current time as the start time for uptime calculation and sets the node type.
//
// Parameters:
//   - nodeType: A string indicating the type of the node (e.g., "main", "follower").
//
// Returns:
//   - A pointer to the newly created MetricsServer.
func NewMetricsServer(nodeType string) *MetricsServer {
	return &MetricsServer{
		startTime: time.Now(),
		NodeType:  nodeType,
	}
}

// GetSystemMetrics is the gRPC method implementation that gathers and returns current system metrics.
// It uses the gopsutil library to get CPU and memory statistics, os.Hostname for the hostname,
// and calculates uptime based on the server's start time.
// The method also includes the NodeType provided during server initialization.
//
// Note: The code contains commented-out OpenTelemetry tracing calls ("manual" way),
// which could be re-enabled for distributed tracing if needed.
//
// Parameters:
//   - ctx: The context.Context for the RPC call (currently unused beyond potential tracing).
//   - _ : The pb.SystemMetricsRequest, which is currently empty and not used.
//
// Returns:
//   - A pointer to a pb.SystemMetricsResponse containing the collected metrics.
//   - An error, which is always nil in the current implementation as data retrieval errors
//     from gopsutil or os.Hostname are typically ignored (defaulting to zero values or empty strings).
func (s *MetricsServer) GetSystemMetrics(ctx context.Context, _ *pb.SystemMetricsRequest) (*pb.SystemMetricsResponse, error) {
	// "manual" way (OpenTelemetry tracing - currently commented out)
	// tracer := otel.Tracer("deadalus.system.metrics")
	// _, span := tracer.Start(ctx, "GetSystemMetrics")
	// defer span.End()
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
