package jobworker

import (
	"context"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/server/common"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/jobworker"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
)

type JobWorkerService struct {
	pb.UnimplementedJobWorkerServiceServer
	startTime   time.Time
	Config      *common.ServerConfing
	JobWorkerBO *bo.JobWorkerBO
}

func NewJobWorkerService(config *common.ServerConfing) *JobWorkerService {
	return &JobWorkerService{
		startTime:   time.Now(),
		Config:      config,
		JobWorkerBO: bo.NewJobWorkerBO(config),
	}
}

func (s *JobWorkerService) ClaimWork(ctx context.Context, r *pb.ClaimWorkRequest) (*pb.ClaimWorkResponse, error) {
	// Map proto capacity policies to model (keyed by index as string)
	capacityPolicies := make(map[string]models.ClaimWorkCapacityPolicy, len(r.CapacityPolicies))
	for i, cp := range r.CapacityPolicies {
		policy := models.ClaimWorkCapacityPolicy{
			MaxQueueMessages:     int(cp.MaxQueueMessages),
			CurrentQueueMessages: int(cp.CurrentQueueMessages),
		}
		if cp.ClaimWorkFilter != nil {
			policy.ClaimWorkFilter = models.ClaimWorkFilter{
				TenantCodes:               cp.ClaimWorkFilter.TenantCodes,
				ExcludeTenantCodes:        cp.ClaimWorkFilter.ExcludeTenantCodes,
				TenantPatterns:            cp.ClaimWorkFilter.TenantPatterns,
				ExcludeTenantPatterns:     cp.ClaimWorkFilter.ExcludeTenantPatterns,
				VNamespaces:               cp.ClaimWorkFilter.VNamespaces,
				ExcludeVNamespaces:        cp.ClaimWorkFilter.ExcludeVNamespaces,
				VNamespacePatterns:        cp.ClaimWorkFilter.VNamespacePatterns,
				ExcludeVNamespacePatterns: cp.ClaimWorkFilter.ExcludeVNamespacePatterns,
				QueueCodes:                cp.ClaimWorkFilter.QueueCodes,
				ExcludeQueueCodes:         cp.ClaimWorkFilter.ExcludeQueueCodes,
				QueuePatterns:             cp.ClaimWorkFilter.QueuePatterns,
				ExcludeQueuePatterns:      cp.ClaimWorkFilter.ExcludeQueuePatterns,
			}
		}
		capacityPolicies[fmt.Sprintf("policy-%d", i)] = policy
	}

	_, err := s.JobWorkerBO.ClaimWork(ctx, r.WorkerID, r.WorkerName, r.Information, capacityPolicies)
	if err != nil {
		return nil, err
	}

	return &pb.ClaimWorkResponse{
		Knowledge: "ok",
	}, nil
}
