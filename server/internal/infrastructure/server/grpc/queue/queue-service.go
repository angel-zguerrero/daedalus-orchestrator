package queue

import (
	"context"
	"fmt"
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

// Helper functions to convert between protobuf and model types
func convertDesiredPriorityThresholds(protoMap map[int32]int32) map[int]int {
	if protoMap == nil {
		return nil
	}
	result := make(map[int]int)
	for k, v := range protoMap {
		result[int(k)] = int(v)
	}
	return result
}

func convertPriorityThresholdsToProto(modelMap map[int]int) map[int32]int32 {
	if modelMap == nil {
		return nil
	}
	result := make(map[int32]int32)
	for k, v := range modelMap {
		result[int32(k)] = int32(v)
	}
	return result
}

func (s *QueueService) CreateQueue(ctx context.Context, r *pb.CreateQueueRequest) (*pb.CreateQueueResponse, error) {
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantId)
	if err != nil {
		return nil, err
	}

	// Validate queue type
	if !isValidQueueType(r.Type) {
		return nil, fmt.Errorf("invalid queue type: %s. Valid types are: standard, delayed, dead-letter", r.Type)
	}

	// Create queue with new properties
	queue := &models.Queue{
		Code:                      r.Code,
		VNamespace:                r.Vnamespace,
		Name:                      r.Name,
		Type:                      models.QueueType(r.Type),
		State:                     models.QueueActive, // Default state
		TTLQueue:                  int(r.TtlQueue),
		AllowDuplicated:           r.AllowDuplicated,
		MaxAttempts:               int(r.MaxAttempts),
		DesiredPriorityThresholds: convertDesiredPriorityThresholds(r.DesiredPriorityThresholds),
	}

	// Set defaults if not provided
	if queue.MaxAttempts == 0 {
		queue.MaxAttempts = 1
	}

	queuesResult, err := s.QueueBO.BulkCreateQueue(ctx, []*models.Queue{queue}, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		return nil, err
	}

	result := queuesResult[0]

	return &pb.CreateQueueResponse{
		Message: "Queue was asserted",
		Result: &pb.Queue{
			Id:                        result.ID,
			Code:                      result.Code,
			Name:                      result.Name,
			Type:                      string(result.Type),
			State:                     string(result.State),
			Vnamespace:                result.VNamespace,
			CreatedAt:                 result.CreatedAt.Format(time.RFC3339),
			UpdatedAt:                 result.UpdatedAt.Format(time.RFC3339),
			TtlQueue:                  int32(result.TTLQueue),
			AllowDuplicated:           result.AllowDuplicated,
			MaxAttempts:               int32(result.MaxAttempts),
			DesiredPriorityThresholds: convertPriorityThresholdsToProto(result.DesiredPriorityThresholds),
			PriorityThresholds:        convertPriorityThresholdsToProto(result.PriorityThresholds),
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
		// Validate queue type
		if !isValidQueueType(t.Type) {
			return nil, fmt.Errorf("invalid queue type: %s. Valid types are: standard, delayed, dead-letter", t.Type)
		}

		queue := &models.Queue{
			Code:                      t.Code,
			VNamespace:                t.Vnamespace,
			Name:                      t.Name,
			Type:                      models.QueueType(t.Type),
			State:                     models.QueueState(t.State),
			TTLQueue:                  int(t.TtlQueue),
			AllowDuplicated:           t.AllowDuplicated,
			MaxAttempts:               int(t.MaxAttempts),
			DesiredPriorityThresholds: convertDesiredPriorityThresholds(t.DesiredPriorityThresholds),
		}
		// Set defaults if not provided
		if queue.MaxAttempts == 0 {
			queue.MaxAttempts = 1
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
			Id:                        e.ID,
			Code:                      e.Code,
			Name:                      e.Name,
			Type:                      string(e.Type),
			State:                     string(e.State),
			Vnamespace:                e.VNamespace,
			CreatedAt:                 e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:                 e.UpdatedAt.Format(time.RFC3339),
			TtlQueue:                  int32(e.TTLQueue),
			AllowDuplicated:           e.AllowDuplicated,
			MaxAttempts:               int32(e.MaxAttempts),
			DesiredPriorityThresholds: convertPriorityThresholdsToProto(e.DesiredPriorityThresholds),
			PriorityThresholds:        convertPriorityThresholdsToProto(e.PriorityThresholds),
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
			Id:                        queue.ID,
			Code:                      queue.Code,
			Name:                      queue.Name,
			Type:                      string(queue.Type),
			State:                     string(queue.State),
			Vnamespace:                queue.VNamespace,
			CreatedAt:                 queue.CreatedAt.Format(time.RFC3339),
			UpdatedAt:                 queue.UpdatedAt.Format(time.RFC3339),
			TtlQueue:                  int32(queue.TTLQueue),
			AllowDuplicated:           queue.AllowDuplicated,
			MaxAttempts:               int32(queue.MaxAttempts),
			DesiredPriorityThresholds: convertPriorityThresholdsToProto(queue.DesiredPriorityThresholds),
			PriorityThresholds:        convertPriorityThresholdsToProto(queue.PriorityThresholds),
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
			Id:                        e.ID,
			Code:                      e.Code,
			Name:                      e.Name,
			Type:                      string(e.Type),
			State:                     string(e.State),
			Vnamespace:                e.VNamespace,
			CreatedAt:                 e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:                 e.UpdatedAt.Format(time.RFC3339),
			TtlQueue:                  int32(e.TTLQueue),
			AllowDuplicated:           e.AllowDuplicated,
			MaxAttempts:               int32(e.MaxAttempts),
			DesiredPriorityThresholds: convertPriorityThresholdsToProto(e.DesiredPriorityThresholds),
			PriorityThresholds:        convertPriorityThresholdsToProto(e.PriorityThresholds),
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

// isValidQueueType validates if the queue type is one of the allowed types
func isValidQueueType(queueType string) bool {
	switch queueType {
	case "standard", "delayed", "dead-letter":
		return true
	default:
		return false
	}
}
