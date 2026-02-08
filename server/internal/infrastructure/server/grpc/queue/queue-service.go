package queue

import (
	"context"
	"fmt"
	"time"

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
}

func NewQueueService(config *common.ServerConfing) *QueueService {
	return &QueueService{
		Config:  config,
		QueueBO: bo.NewQueueBO(config),
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
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	// Validate queue type
	if !isValidQueueType(r.Type) {
		return nil, fmt.Errorf("invalid queue type: %s. Valid types are: standard, delayed", r.Type)
	}

	// Create queue with new properties
	queue := &models.Queue{
		Code:                                  r.Code,
		VNamespace:                            r.Vnamespace,
		Name:                                  r.Name,
		Type:                                  models.QueueType(r.Type),
		State:                                 models.QueueActive, // Default state
		DefaultQueueMessageTTL:                int(r.DefaultQueueMessageTTL),
		DefaultQueueMessageDelayTime:          int(r.DefaultQueueMessageDelayTime),
		QueueExpires:                          int(r.QueueExpires),
		AllowDuplicated:                       r.AllowDuplicated,
		MaxAttempts:                           int(r.MaxAttempts),
		MaxQueueSize:                          int(r.MaxQueueSize),
		DesiredPriorityThresholds:             convertDesiredPriorityThresholds(r.DesiredPriorityThresholds),
		Headers:                               r.Headers,
		DeadLetterExchangeId:                  r.DeadLetterExchangeId,
		DeadLetterExchangeRoutingKeyOrPattern: r.DeadLetterExchangeRoutingKeyOrPattern,
	}

	// Set defaults if not provided
	if queue.MaxAttempts == 0 {
		queue.MaxAttempts = 1
	}

	queuesResult, err := s.QueueBO.BulkCreateQueue(ctx, []*models.Queue{queue}, cf, cfs, tenant, tenantNode)
	if err != nil {
		return nil, err
	}

	result := queuesResult[0]

	return &pb.CreateQueueResponse{
		Message: "Queue was asserted",
		Result: &pb.Queue{
			Id:                                    result.ID,
			Code:                                  result.Code,
			Name:                                  result.Name,
			Type:                                  string(result.Type),
			State:                                 string(result.State),
			Vnamespace:                            result.VNamespace,
			CreatedAt:                             result.CreatedAt.Format(time.RFC3339),
			UpdatedAt:                             result.UpdatedAt.Format(time.RFC3339),
			DefaultQueueMessageTTL:                int32(result.DefaultQueueMessageTTL),
			DefaultQueueMessageDelayTime:          int32(result.DefaultQueueMessageDelayTime),
			QueueExpires:                          int32(result.QueueExpires),
			AllowDuplicated:                       result.AllowDuplicated,
			MaxAttempts:                           int32(result.MaxAttempts),
			MaxQueueSize:                          int32(result.MaxQueueSize),
			MessagesCount:                         int32(result.MessagesCount),
			DesiredPriorityThresholds:             convertPriorityThresholdsToProto(result.DesiredPriorityThresholds),
			PriorityThresholds:                    convertPriorityThresholdsToProto(result.PriorityThresholds),
			Headers:                               result.Headers,
			DeadLetterExchangeId:                  result.DeadLetterExchangeId,
			DeadLetterExchangeRoutingKeyOrPattern: result.DeadLetterExchangeRoutingKeyOrPattern,
		},
	}, nil
}

func (s *QueueService) BulkCreateQueue(ctx context.Context, r *pb.BulkCreateQueueRequest) (*pb.BulkCreateQueueResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	queues := []*models.Queue{}
	for _, t := range r.Queues {
		// Validate queue type
		if !isValidQueueType(t.Type) {
			return nil, fmt.Errorf("invalid queue type: %s. Valid types are: standard, delayed", t.Type)
		}

		queue := &models.Queue{
			Code:                                  t.Code,
			VNamespace:                            t.Vnamespace,
			Name:                                  t.Name,
			Type:                                  models.QueueType(t.Type),
			State:                                 models.QueueState(t.State),
			DefaultQueueMessageTTL:                int(t.DefaultQueueMessageTTL),
			DefaultQueueMessageDelayTime:          int(t.DefaultQueueMessageDelayTime),
			QueueExpires:                          int(t.QueueExpires),
			AllowDuplicated:                       t.AllowDuplicated,
			MaxAttempts:                           int(t.MaxAttempts),
			MaxQueueSize:                          int(t.MaxQueueSize),
			DesiredPriorityThresholds:             convertDesiredPriorityThresholds(t.DesiredPriorityThresholds),
			Headers:                               t.Headers,
			DeadLetterExchangeId:                  t.DeadLetterExchangeId,
			DeadLetterExchangeRoutingKeyOrPattern: t.DeadLetterExchangeRoutingKeyOrPattern,
		}
		// Set defaults if not provided
		if queue.MaxAttempts == 0 {
			queue.MaxAttempts = 1
		}
		queues = append(queues, queue)
	}

	queuesResult, err := s.QueueBO.BulkCreateQueue(ctx, queues, cf, cfs, tenant, tenantNode)
	if err != nil {
		return nil, err
	}

	rQueues := []*pb.Queue{}
	for _, e := range queuesResult {
		ex := &pb.Queue{
			Id:                                    e.ID,
			Code:                                  e.Code,
			Name:                                  e.Name,
			Type:                                  string(e.Type),
			State:                                 string(e.State),
			Vnamespace:                            e.VNamespace,
			CreatedAt:                             e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:                             e.UpdatedAt.Format(time.RFC3339),
			DefaultQueueMessageTTL:                int32(e.DefaultQueueMessageTTL),
			DefaultQueueMessageDelayTime:          int32(e.DefaultQueueMessageDelayTime),
			QueueExpires:                          int32(e.QueueExpires),
			AllowDuplicated:                       e.AllowDuplicated,
			MaxAttempts:                           int32(e.MaxAttempts),
			MaxQueueSize:                          int32(e.MaxQueueSize),
			MessagesCount:                         int32(e.MessagesCount),
			DesiredPriorityThresholds:             convertPriorityThresholdsToProto(e.DesiredPriorityThresholds),
			PriorityThresholds:                    convertPriorityThresholdsToProto(e.PriorityThresholds),
			Headers:                               e.Headers,
			DeadLetterExchangeId:                  e.DeadLetterExchangeId,
			DeadLetterExchangeRoutingKeyOrPattern: e.DeadLetterExchangeRoutingKeyOrPattern,
		}
		rQueues = append(rQueues, ex)
	}

	return &pb.BulkCreateQueueResponse{
		Message: "Queues were asserted",
		Result:  rQueues,
	}, nil
}

func (s *QueueService) GetQueue(ctx context.Context, r *pb.GetQueueRequest) (*pb.GetQueueResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	queue, err := s.QueueBO.GetQueue(ctx, r.Code, r.Vnamespace, false, cf, cfs, tenant, tenantNode)
	if err != nil {
		return nil, err
	}

	return &pb.GetQueueResponse{
		Message: "Queue",
		Result: &pb.Queue{
			Id:                                    queue.ID,
			Code:                                  queue.Code,
			Name:                                  queue.Name,
			Type:                                  string(queue.Type),
			State:                                 string(queue.State),
			Vnamespace:                            queue.VNamespace,
			CreatedAt:                             queue.CreatedAt.Format(time.RFC3339),
			UpdatedAt:                             queue.UpdatedAt.Format(time.RFC3339),
			DefaultQueueMessageTTL:                int32(queue.DefaultQueueMessageTTL),
			DefaultQueueMessageDelayTime:          int32(queue.DefaultQueueMessageDelayTime),
			QueueExpires:                          int32(queue.QueueExpires),
			AllowDuplicated:                       queue.AllowDuplicated,
			MaxAttempts:                           int32(queue.MaxAttempts),
			MaxQueueSize:                          int32(queue.MaxQueueSize),
			MessagesCount:                         int32(queue.MessagesCount),
			DesiredPriorityThresholds:             convertPriorityThresholdsToProto(queue.DesiredPriorityThresholds),
			PriorityThresholds:                    convertPriorityThresholdsToProto(queue.PriorityThresholds),
			Headers:                               queue.Headers,
			DeadLetterExchangeId:                  queue.DeadLetterExchangeId,
			DeadLetterExchangeRoutingKeyOrPattern: queue.DeadLetterExchangeRoutingKeyOrPattern,
		},
	}, nil
}

func (s *QueueService) GetQueues(ctx context.Context, r *pb.GetQueuesRequest) (*pb.GetQueuesResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	findResult, err := s.QueueBO.GetQueues(ctx, r.Q, r.Cursor, int(r.PageSize), r.Vnamespace, r.IncludeHeaders, cf, cfs, tenant, tenantNode, true)
	if err != nil {
		return nil, err
	}

	rQueues := []*pb.Queue{}
	for _, e := range findResult.Entities {
		ex := &pb.Queue{
			Id:                                    e.ID,
			Code:                                  e.Code,
			Name:                                  e.Name,
			Type:                                  string(e.Type),
			State:                                 string(e.State),
			Vnamespace:                            e.VNamespace,
			CreatedAt:                             e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:                             e.UpdatedAt.Format(time.RFC3339),
			DefaultQueueMessageTTL:                int32(e.DefaultQueueMessageTTL),
			DefaultQueueMessageDelayTime:          int32(e.DefaultQueueMessageDelayTime),
			QueueExpires:                          int32(e.QueueExpires),
			AllowDuplicated:                       e.AllowDuplicated,
			MaxAttempts:                           int32(e.MaxAttempts),
			MaxQueueSize:                          int32(e.MaxQueueSize),
			MessagesCount:                         int32(e.MessagesCount),
			DesiredPriorityThresholds:             convertPriorityThresholdsToProto(e.DesiredPriorityThresholds),
			PriorityThresholds:                    convertPriorityThresholdsToProto(e.PriorityThresholds),
			DeadLetterExchangeId:                  e.DeadLetterExchangeId,
			DeadLetterExchangeRoutingKeyOrPattern: e.DeadLetterExchangeRoutingKeyOrPattern,
		}

		// Add headers if requested and available
		if r.IncludeHeaders && e.Headers != nil {
			ex.Headers = e.Headers
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
	// Usar el tenant context inyectado por el interceptor en lugar de obtenerlo manualmente
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	err := s.QueueBO.DeleteQueue(ctx, r.Code, r.Vnamespace, cf, cfs, tenant, tenantNode)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteQueueResponse{
		Message: "Queue " + r.Code + " in namespace " + r.Vnamespace + " was deleted",
	}, nil
}

func (s *QueueService) EnqueueMessage(ctx context.Context, r *pb.EnqueueMessageRequest) (*pb.EnqueueMessageResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	// Create the message
	message := models.QueueMessage{
		Content:     []byte(r.Content),
		ContentType: r.ContentType,
		Headers:     r.Headers,
		Priority:    int(r.Priority),
		Handler:     r.Handler,
		Parameters:  r.Parameters,
		VNamespace:  r.Vnamespace,
	}

	// Enqueue the message
	messageID, err := s.QueueBO.EnqueueMessage(ctx, r.QueueCode, message, r.Vnamespace, cf, cfs, tenant, tenantNode)
	if err != nil {
		return nil, err
	}

	// Build result map
	result := make(map[string]string)
	result[r.QueueCode] = messageID

	return &pb.EnqueueMessageResponse{
		Message:   "Message enqueued successfully",
		MessageId: messageID,
		Result:    result,
	}, nil
}

// isValidQueueType validates if the queue type is one of the allowed types
func isValidQueueType(queueType string) bool {
	switch queueType {
	case "standard":
		return true
	default:
		return false
	}
}
