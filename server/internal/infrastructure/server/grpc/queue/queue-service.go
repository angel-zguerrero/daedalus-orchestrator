package queue

import (
	"context"
	"strconv"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/queue"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
)

type QueueService struct {
	pb.UnimplementedQueueServiceServer
	startTime time.Time
	Config    *common.ServerConfing
	QueueBO   *bo.QueueBO
	TenantBO  *bo.TenantBO
}

func NewQueueService(config *common.ServerConfing) *QueueService {
	return &QueueService{
		Config:   config,
		QueueBO:  bo.NewQueueBO(config),
		TenantBO: bo.NewTenantBO(config),
	}
}

func (s *QueueService) CreateQueue(ctx context.Context, r *pb.CreateQueueRequest) (*pb.CreateQueueResponse, error) {
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantId)
	if err != nil {
		return nil, err
	}

	queue, err := s.QueueBO.CreateQueue(ctx, r.Code, r.Vnamespace, r.Name, models.QueueType(r.Type), db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		return nil, err
	}

	return &pb.CreateQueueResponse{
		Message: "Queue was asserted",
		Result: &pb.Queue{
			ID:         queue.ID,
			Code:       queue.Code,
			Name:       queue.Name,
			Type:       string(queue.Type),
			State:      string(queue.State),
			VNamespace: queue.VNamespace,
			CreatedAt:  queue.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  queue.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *QueueService) BulkCreateQueue(ctx context.Context, r *pb.BulkCreateQueueRequest) (*pb.BulkCreateQueueResponse, error) {
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantId)
	if err != nil {
		return nil, err
	}

	queues := []*models.Queue{}
	for _, t := range r.Queues {
		queue := &models.Queue{
			Code:       t.Code,
			VNamespace: t.Vnamespace,
			Name:       t.Name,
			Type:       models.QueueType(t.Type),
			State:      models.QueueState(t.State),
		}
		queues = append(queues, queue)
	}

	queuesResult, err := s.QueueBO.BulkCreateQueue(ctx, queues, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		return nil, err
	}

	rQueues := []*pb.Queue{}
	for _, e := range queuesResult {
		ex := &pb.Queue{
			ID:         e.ID,
			Code:       e.Code,
			Name:       e.Name,
			Type:       string(e.Type),
			State:      string(e.State),
			VNamespace: e.VNamespace,
			CreatedAt:  e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  e.UpdatedAt.Format(time.RFC3339),
		}
		rQueues = append(rQueues, ex)
	}

	return &pb.BulkCreateQueueResponse{
		Message: "Queues were asserted",
		Result:  rQueues,
	}, nil
}

func (s *QueueService) GetQueue(ctx context.Context, r *pb.GetQueueRequest) (*pb.GetQueueResponse, error) {
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantId)
	if err != nil {
		return nil, err
	}

	queue, err := s.QueueBO.GetQueue(ctx, r.QueueId, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		return nil, err
	}

	return &pb.GetQueueResponse{
		Message: "Queue",
		Result: &pb.Queue{
			ID:         queue.ID,
			Code:       queue.Code,
			Name:       queue.Name,
			Type:       string(queue.Type),
			State:      string(queue.State),
			VNamespace: queue.VNamespace,
			CreatedAt:  queue.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  queue.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *QueueService) GetQueues(ctx context.Context, r *pb.GetQueuesRequest) (*pb.GetQueuesResponse, error) {
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantId)
	if err != nil {
		return nil, err
	}

	findResult, err := s.QueueBO.GetQueues(ctx, r.Q, r.Cursor, int(r.PageSize), r.Vnamespace, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		return nil, err
	}

	rQueues := []*pb.Queue{}
	for _, e := range findResult.Entities {
		ex := &pb.Queue{
			ID:         e.ID,
			Code:       e.Code,
			Name:       e.Name,
			Type:       string(e.Type),
			State:      string(e.State),
			VNamespace: e.VNamespace,
			CreatedAt:  e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  e.UpdatedAt.Format(time.RFC3339),
		}
		rQueues = append(rQueues, ex)
	}

	return &pb.GetQueuesResponse{
		Message: "Queue list",
		Result: &pb.QueueFindResult{
			Entities: rQueues,
			Cursor:   findResult.Cursor,
		},
	}, nil
}

func (s *QueueService) DeleteQueue(ctx context.Context, r *pb.DeleteQueueRequest) (*pb.DeleteQueueResponse, error) {
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantId)
	if err != nil {
		return nil, err
	}

	err = s.QueueBO.DeleteQueue(ctx, r.QueueId, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteQueueResponse{
		Message: "Queue " + r.QueueId + " was deleted",
	}, nil
}
