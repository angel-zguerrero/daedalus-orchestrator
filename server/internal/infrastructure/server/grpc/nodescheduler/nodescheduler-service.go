package nodescheduler

import (
	"context"
	"time"

	"deadalus-orch/server/internal/infrastructure/server/common"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/nodescheduler"
	bo "deadalus-orch/server/internal/usecase/business-logic"
)

type NodeSchedulerService struct {
	pb.UnimplementedNodeSchedulerServiceServer
	startTime       time.Time
	Config          *common.ServerConfing
	NodeSchedulerBO *bo.NodeSchedulerBO
}

func NewNodeSchedulerService(config *common.ServerConfing) *NodeSchedulerService {
	return &NodeSchedulerService{
		startTime:       time.Now(),
		Config:          config,
		NodeSchedulerBO: bo.NewNodeSchedulerBO(config),
	}
}

func (s *NodeSchedulerService) GetNodeSchedulers(ctx context.Context, r *pb.GetNodeSchedulersRequest) (*pb.GetNodeSchedulersResponse, error) {
	findResult, err := s.NodeSchedulerBO.GetNodeSchedulers(ctx, r.Q, r.Cursor, int(r.PageSize))
	if err != nil {
		return nil, err
	}

	nodeSchedulers := make([]*pb.NodeScheduler, len(findResult.Entities))
	for i, ns := range findResult.Entities {
		// Ensure proper type conversion and initialization
		information := make(map[string]string)
		if ns.Information != nil {
			for k, v := range ns.Information {
				information[k] = v
			}
		}

		nodeSchedulers[i] = &pb.NodeScheduler{
			ID:               ns.ID,
			Name:             ns.Name,
			TTL:              int64(ns.TTL), // Explicit conversion to int64
			LastHeartbeat:    ns.LastHeartbeat.Format(time.RFC3339),
			Information:      information,
			ConnectionStatus: string(ns.ConnectionStatus),
			CreatedAt:        ns.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        ns.UpdatedAt.Format(time.RFC3339),
			// AssignedTenantNodeIndex: int32(ns.AssignedTenantNodeIndex),
			// RunningStatus:           string(ns.RunningStatus),
		}
	}

	return &pb.GetNodeSchedulersResponse{
		Message: "NodeScheduler list",
		Result: &pb.NodeSchedulerFindResult{
			Entities: nodeSchedulers,
			Cursor:   findResult.Cursor,
		},
	}, nil
}

func (s *NodeSchedulerService) GetNodeScheduler(ctx context.Context, r *pb.GetNodeSchedulerRequest) (*pb.GetNodeSchedulerResponse, error) {
	nodeScheduler, err := s.NodeSchedulerBO.GetNodeScheduler(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	// Ensure proper type conversion and initialization
	information := make(map[string]string)
	if nodeScheduler.Information != nil {
		for k, v := range nodeScheduler.Information {
			information[k] = v
		}
	}

	return &pb.GetNodeSchedulerResponse{
		Message: "NodeScheduler",
		Result: &pb.NodeScheduler{
			ID:               nodeScheduler.ID,
			Name:             nodeScheduler.Name,
			TTL:              int64(nodeScheduler.TTL), // Explicit conversion to int64
			LastHeartbeat:    nodeScheduler.LastHeartbeat.Format(time.RFC3339),
			Information:      information,
			ConnectionStatus: string(nodeScheduler.ConnectionStatus),
			CreatedAt:        nodeScheduler.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        nodeScheduler.UpdatedAt.Format(time.RFC3339),
			// AssignedTenantNodeIndex: int32(nodeScheduler.AssignedTenantNodeIndex),
			// RunningStatus:           string(nodeScheduler.RunningStatus),
		},
	}, nil
}
