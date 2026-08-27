package jobworker

import (
	"context"
	"fmt"
	"io"
	"time"

	"deadalus-orch/server/internal/infrastructure/server/common"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/jobworker"
	"deadalus-orch/server/internal/infrastructure/server/grpc/buffer"
	configPkg "deadalus-orch/server/internal/pkg/config"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
)

type JobWorkerService struct {
	pb.UnimplementedJobWorkerServiceServer
	startTime      time.Time
	Config         *common.ServerConfing
	JobWorkerBO    *bo.JobWorkerBO
	ackBuffer      *buffer.MessageBuffer[buffer.AckBufferedMessage]
	deliveredBuffer *buffer.MessageBuffer[buffer.DeliveredBufferedMessage]
}

func NewJobWorkerService(config *common.ServerConfing) *JobWorkerService {
	ackBuffer := buffer.NewMessageBuffer[buffer.AckBufferedMessage](
		time.Duration(config.PublishBufferFlushIntervalMs)*time.Millisecond,
		config.PublishBufferMaxSize,
		config.Logger,
		buffer.NewAckFlusher(config.Logger, configPkg.GlobalConfiguration.ApiRaftTimeout),
	)
	
	deliveredBuffer := buffer.NewMessageBuffer[buffer.DeliveredBufferedMessage](
		time.Duration(config.PublishBufferFlushIntervalMs)*time.Millisecond,
		config.PublishBufferMaxSize,
		config.Logger,
		buffer.NewDeliveredFlusher(config.Logger, configPkg.GlobalConfiguration.ApiRaftTimeout),
	)

	dequeueBuffer := buffer.NewMessageBuffer[buffer.DequeueBufferedMessage](
		time.Duration(config.PublishBufferFlushIntervalMs)*time.Millisecond,
		config.PublishBufferMaxSize,
		config.Logger,
		buffer.NewDequeueFlusher(config.Logger, configPkg.GlobalConfiguration.ApiRaftTimeout),
	)

	dequeueFunc := func(ctx context.Context, req bo.DequeueRequestMessage) (bo.DequeueResponseMessage, error) {
		resChan := make(chan buffer.DequeueConfirmation, 1)
		dequeueBuffer.Add(ctx, buffer.DequeueBufferedMessage{
			QueueID:                      req.QueueID,
			JobWorkerID:                  req.JobWorkerID,
			LeaseDuration:                req.LeaseDuration,
			JobWorkerCapacityPolicyIndex: req.JobWorkerCapacityPolicyIndex,
			CF:                           req.CF,
			CFS:                          req.CFS,
			TenantNode:                   req.TenantNode,
			ResponseChan:                 resChan,
		})

		select {
		case conf := <-resChan:
			return bo.DequeueResponseMessage{Result: conf.Result}, conf.Error
		case <-ctx.Done():
			return bo.DequeueResponseMessage{}, ctx.Err()
		}
	}

	heartbeatBuffer := buffer.NewMessageBuffer[buffer.JobWorkerHeartbeatBufferedMessage](
		time.Duration(config.PublishBufferFlushIntervalMs)*time.Millisecond,
		config.PublishBufferMaxSize,
		config.Logger,
		buffer.NewJobWorkerHeartbeatFlusher(config.Logger, configPkg.GlobalConfiguration.ApiRaftTimeout),
	)

	heartbeatFunc := func(ctx context.Context, worker models.JobWorker, masterNode *dragonboat.RaftNode) error {
		resChan := make(chan buffer.HeartbeatConfirmation, 1)
		heartbeatBuffer.Add(ctx, buffer.JobWorkerHeartbeatBufferedMessage{
			JobWorker:    worker,
			MasterNode:   masterNode,
			ResponseChan: resChan,
		})
		go func(wID string, rc chan buffer.HeartbeatConfirmation) {
			res := <-rc
			if !res.Success {
				config.Logger.Error().Err(res.Error).Str("workerID", wID).Msg("❌ Failed to upsert job worker heartbeat")
			}
		}(worker.ID, resChan)
		return nil
	}

	return &JobWorkerService{
		startTime:       time.Now(),
		Config:          config,
		JobWorkerBO:     bo.NewJobWorkerBO(config, dequeueFunc, heartbeatFunc),
		ackBuffer:       ackBuffer,
		deliveredBuffer: deliveredBuffer,
	}
}

func (s *JobWorkerService) ClaimWork(stream pb.JobWorkerService_ClaimWorkServer) error {
	// Create a channel to receive claimed messages from business logic
	messageChan := make(chan bo.ClaimedMessage, 100)

	// Track if we've sent the initial ACK
	ackSent := false

	// workerID is captured from the first request and used for cursor cleanup on disconnect.
	workerID := ""

	// Goroutine to receive claim requests from client
	requestChan := make(chan *pb.ClaimWorkRequest, 10)
	errChan := make(chan error, 1)

	go func() {
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				close(requestChan)
				return
			}
			if err != nil {
				errChan <- err
				close(requestChan)
				return
			}
			requestChan <- req
		}
	}()

	// Main loop: process requests and send messages
	for {
		select {
		case req, ok := <-requestChan:
			if !ok {
				// Client closed the stream.
				return nil
			}

			// Send ACK on first request and capture workerID for lifecycle management.
			if !ackSent {
				workerID = req.WorkerID
				ackMsg := &pb.ClaimWorkStreamMessage{
					Message: &pb.ClaimWorkStreamMessage_Ack{
						Ack: &pb.ClaimWorkResponse{
							Knowledge: "ok",
						},
					},
				}
				if err := stream.Send(ackMsg); err != nil {
					return fmt.Errorf("failed to send ack: %w", err)
				}
				ackSent = true
			}

			// Map proto capacity policies to model
			capacityPolicies := make(map[string]models.ClaimWorkCapacityPolicy, len(req.CapacityPolicies))
			for i, cp := range req.CapacityPolicies {
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

			// Process the claim work request

			err := s.JobWorkerBO.ClaimWork(stream.Context(), req.WorkerID, req.WorkerName, req.Information, capacityPolicies, messageChan)
			if err != nil {
				s.Config.Logger.Error().Err(err).Msg("Failed to process claim work")
				// Don't return error, continue processing
			}

		case claimed, ok := <-messageChan:
			if !ok {
				// Channel closed, shouldn't happen but handle gracefully
				return nil
			}

			// Convert to protobuf message
			pbMsg := &pb.QueueMessage{
				ID:          claimed.Message.ID,
				MessageID:   claimed.Message.MessageID,
				Content:     string(claimed.Message.Content),
				ContentType: claimed.Message.ContentType,
				Headers:     claimed.Message.Headers,
				QueueID:     claimed.Message.QueueID,
				Priority:    int32(claimed.Message.Priority),
				Attempts:    int32(claimed.Message.Attempts),
				Handler:     claimed.Message.Handler,
				Parameters:  claimed.Message.Parameters,
				VNamespace:  claimed.Message.VNamespace,
				CreatedAt:   claimed.Message.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}

			pbLease := &pb.QueueMessageLease{
				ID:             claimed.Lease.ID,
				QueueMessageID: claimed.Lease.QueueMessageID,
				WorkerID:       claimed.Lease.WorkerID,
				LeaseUntil:     claimed.Lease.LeaseUntil.Format("2006-01-02T15:04:05Z07:00"),
			}

			claimedMsgProto := &pb.ClaimedQueueMessage{
				Message:                  pbMsg,
				Lease:                    pbLease,
				TenantCode:               claimed.TenantCode,
				CapacityPolicyIndexMatch: int32(claimed.Lease.JobWorkerCapacityPolicyIndexMatch),
			}

			streamMsg := &pb.ClaimWorkStreamMessage{
				Message: &pb.ClaimWorkStreamMessage_ClaimedMessage{
					ClaimedMessage: claimedMsgProto,
				},
			}

			if err := stream.Send(streamMsg); err != nil {
				s.Config.Logger.Error().Err(err).Msg("Failed to send message to stream")
				return err
			}
			
			// Mark lease as delivered via the buffer
			resChan := make(chan buffer.DeliveredConfirmation, 1)
			s.deliveredBuffer.Add(stream.Context(), buffer.DeliveredBufferedMessage{
				LeaseID:      claimed.Lease.ID,
				CF:           claimed.CF,
				CFS:          claimed.CFS,
				TenantNode:   claimed.TenantNode,
				ResponseChan: resChan,
			})
			
			// Fire and forget, or log errors asynchronously
			go func(lID string, rc chan buffer.DeliveredConfirmation) {
				res := <-rc
				if !res.Success {
					s.Config.Logger.Error().Err(res.Error).Str("leaseID", lID).Msg("❌ Failed to mark lease delivered")
				}
			}(claimed.Lease.ID, resChan)

			s.Config.Logger.Debug().
				Str("messageID", claimed.Message.ID).
				Str("tenant", claimed.TenantCode).
				Msg("Sent message to client")

		case err := <-errChan:
			return fmt.Errorf("error receiving from client: %w", err)

		case <-stream.Context().Done():
			// Client disconnected
			s.Config.Logger.Info().Str("workerID", workerID).Msg("Client disconnected")
			return stream.Context().Err()
		}
	}
}

func (s *JobWorkerService) AckMessage(ctx context.Context, req *pb.AckMessageRequest) (*pb.AckMessageResponse, error) {
	if req.LeaseID == "" {
		return &pb.AckMessageResponse{Success: false, Message: "leaseID is required"}, nil
	}
	if req.TenantCode == "" {
		return &pb.AckMessageResponse{Success: false, Message: "tenantCode is required"}, nil
	}

	tenantBO := bo.NewTenantBO(s.Config)
	tenant, tenantNode, _, err := tenantBO.GetTenant(ctx, req.TenantCode)
	if err != nil {
		s.Config.Logger.Error().Err(err).Msg("Failed to get tenant")
		return &pb.AckMessageResponse{Success: false, Message: err.Error()}, nil
	}
	if tenantNode == nil {
		s.Config.Logger.Error().Str("tenantCode", req.TenantCode).Msg("No node found for tenant")
		return &pb.AckMessageResponse{Success: false, Message: "tenant node not available"}, nil
	}

	cf := db.ColumnFamilyPrefix + fmt.Sprintf("%d", tenant.ColumnFamilyIndex)
	cfs := tenant.ID

	responseChan := make(chan buffer.AckConfirmation, 1)

	bufferedMessage := buffer.AckBufferedMessage{
		LeaseID:      req.LeaseID,
		CF:           cf,
		CFS:          cfs,
		Tenant:       &tenant,
		TenantNode:   tenantNode,
		ResponseChan: responseChan,
	}

	s.ackBuffer.Add(ctx, bufferedMessage)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case confirmation := <-responseChan:
		if confirmation.Error != nil {
			return &pb.AckMessageResponse{
				Success: false,
				Message: confirmation.Error.Error(),
			}, nil
		}
		return &pb.AckMessageResponse{
			Success: true,
			Message: confirmation.Message,
		}, nil
	}
}
