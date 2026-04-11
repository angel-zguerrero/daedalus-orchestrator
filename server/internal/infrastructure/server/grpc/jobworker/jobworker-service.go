package jobworker

import (
	"context"
	"fmt"
	"io"
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

func (s *JobWorkerService) ClaimWork(stream pb.JobWorkerService_ClaimWorkServer) error {
	// Create a channel to receive claimed messages from business logic
	messageChan := make(chan bo.ClaimedMessage, 100)
	defer close(messageChan)

	// Track if we've sent the initial ACK
	ackSent := false

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
				// Client closed the stream
				return nil
			}

			// Send ACK on first request
			if !ackSent {
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
			s.Config.Logger.Debug().
				Str("workerID", req.WorkerID).
				Str("workerName", req.WorkerName).
				Msg("Processing claim work request")

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
				Message:    pbMsg,
				Lease:      pbLease,
				TenantCode: claimed.TenantCode,
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

			s.Config.Logger.Debug().
				Str("messageID", claimed.Message.ID).
				Str("tenant", claimed.TenantCode).
				Msg("Sent message to client")

		case err := <-errChan:
			return fmt.Errorf("error receiving from client: %w", err)

		case <-stream.Context().Done():
			// Client disconnected
			s.Config.Logger.Info().Msg("Client disconnected")
			return stream.Context().Err()
		}
	}
}

func (s *JobWorkerService) AckMessage(ctx context.Context, req *pb.AckMessageRequest) (*pb.AckMessageResponse, error) {
	s.Config.Logger.Debug().
		Str("leaseID", req.LeaseID).
		Str("tenantCode", req.TenantCode).
		Msg("Received AckMessage request")

	err := s.JobWorkerBO.AckMessage(ctx, req.LeaseID, req.TenantCode)
	if err != nil {
		s.Config.Logger.Error().Err(err).Msg("Failed to ack message")
		return &pb.AckMessageResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.AckMessageResponse{
		Success: true,
		Message: "Message acknowledged successfully",
	}, nil
}
