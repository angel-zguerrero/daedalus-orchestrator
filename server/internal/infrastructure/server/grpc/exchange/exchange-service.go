package exchange

import (
	"context"
	"strconv"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/infrastructure/server/grpc/buffer"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/exchange"
	pkg_config "deadalus-orch/server/internal/pkg/config"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
)

type ExchangeService struct {
	pb.UnimplementedExchangeServiceServer
	startTime     time.Time
	Config        *common.ServerConfing
	ExchangeBO    *bo.ExchangeBO
	publishBuffer *buffer.MessageBuffer[buffer.PublishBufferedMessage]
}

func NewExchangeService(config *common.ServerConfing) *ExchangeService {
	exchangeBO := bo.NewExchangeBO(config)
	
	publishBuffer := buffer.NewMessageBuffer[buffer.PublishBufferedMessage](
		time.Duration(config.PublishBufferFlushIntervalMs)*time.Millisecond,
		config.PublishBufferMaxSize,
		config.Logger,
		buffer.NewPublishFlusher(exchangeBO, config.Logger, pkg_config.GlobalConfiguration.ApiRaftTimeout),
	)

	return &ExchangeService{
		startTime:     time.Now(),
		Config:        config,
		ExchangeBO:    exchangeBO,
		publishBuffer: publishBuffer,
	}
}

func (s *ExchangeService) CreateExchange(ctx context.Context, r *pb.CreateExchangeRequest) (*pb.CreateExchangeResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	exchange, err := s.ExchangeBO.CreateExchange(ctx, r.Code, r.Vnamespace, r.Name, models.ExchangeType(r.Type), r.Headers, cf, cfs, tenant, tenantNode)
	if err != nil {
		return nil, err
	}

	return &pb.CreateExchangeResponse{
		Message: "Exchange was asserted",
		Result: &pb.Exchange{
			ID:         exchange.ID,
			Code:       exchange.Code,
			Name:       exchange.Name,
			Type:       string(exchange.Type),
			VNamespace: exchange.VNamespace,
			CreatedAt:  exchange.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  exchange.UpdatedAt.Format(time.RFC3339),
			Headers:    exchange.Headers,
		},
	}, nil
}

func (s *ExchangeService) BulkCreateExchange(ctx context.Context, r *pb.BulkCreateExchangeRequest) (*pb.BulkCreateExchangeResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	exchanges := []*models.Exchange{}
	for _, t := range r.Exchanges {
		exchange := &models.Exchange{
			Code:       t.Code,
			VNamespace: t.Vnamespace,
			Name:       t.Name,
			Type:       models.ExchangeType(t.Type),
			Headers:    t.Headers,
		}
		exchanges = append(exchanges, exchange)
	}

	exchangesResult, err := s.ExchangeBO.BulkCreateExchange(ctx, exchanges, cf, cfs, tenant, tenantNode)
	if err != nil {
		return nil, err
	}

	rExchanges := []*pb.Exchange{}
	for _, e := range exchangesResult {
		ex := &pb.Exchange{
			ID:         e.ID,
			Code:       e.Code,
			Name:       e.Name,
			Type:       string(e.Type),
			VNamespace: e.VNamespace,
			CreatedAt:  e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  e.UpdatedAt.Format(time.RFC3339),
			Headers:    e.Headers,
		}
		rExchanges = append(rExchanges, ex)
	}

	return &pb.BulkCreateExchangeResponse{
		Message: "Exchanges were asserted",
		Result:  rExchanges,
	}, nil
}

func (s *ExchangeService) GetExchange(ctx context.Context, r *pb.GetExchangeRequest) (*pb.GetExchangeResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	exchange, err := s.ExchangeBO.GetExchange(ctx, r.Code, r.Vnamespace, cf, cfs, tenant, tenantNode)
	if err != nil {
		return nil, err
	}

	return &pb.GetExchangeResponse{
		Message: "Exchange",
		Result: &pb.Exchange{
			ID:         exchange.ID,
			Code:       exchange.Code,
			Name:       exchange.Name,
			Type:       string(exchange.Type),
			VNamespace: exchange.VNamespace,
			CreatedAt:  exchange.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  exchange.UpdatedAt.Format(time.RFC3339),
			Headers:    exchange.Headers,
		},
	}, nil
}

func (s *ExchangeService) GetExchanges(ctx context.Context, r *pb.GetExchangesRequest) (*pb.GetExchangesResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	page := int(r.PageSize)
	if page < 2 {
		page = 50
	} else if page > 1000 {
		page = 1000
	}

	findResult, err := s.ExchangeBO.GetExchanges(ctx, r.Q, r.Cursor, page, r.Vnamespace, cf, cfs, tenant, tenantNode)
	if err != nil {
		return nil, err
	}

	exchanges := make([]*pb.Exchange, len(findResult.Entities))
	for i, e := range findResult.Entities {
		exchanges[i] = &pb.Exchange{
			ID:         e.ID,
			Code:       e.Code,
			Name:       e.Name,
			Type:       string(e.Type),
			VNamespace: e.VNamespace,
			CreatedAt:  e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  e.UpdatedAt.Format(time.RFC3339),
			Headers:    e.Headers,
		}
	}

	return &pb.GetExchangesResponse{
		Message: "Exchange list",
		Result: &pb.ExchangeFindResult{
			Entities: exchanges,
			Cursor:   findResult.Cursor,
		},
	}, nil
}

func (s *ExchangeService) DeleteExchange(ctx context.Context, r *pb.DeleteExchangeRequest) (*pb.DeleteExchangeResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	err := s.ExchangeBO.DeleteExchange(ctx, r.Code, r.Vnamespace, cf, cfs, tenant, tenantNode)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteExchangeResponse{
		Message: "Exchange " + r.Code + " in namespace " + r.Vnamespace + " was deleted",
	}, nil
}

func (s *ExchangeService) PublishMessage(ctx context.Context, r *pb.PublishMessageRequest) (*pb.PublishMessageResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	// Convert protobuf message to models.QueueMessage
	message := models.QueueMessage{
		MessageID:     r.Message.MessageId,
		Handler:       r.Message.Handler,
		Priority:      int(r.Message.Priority),
		Parameters:    r.Message.Parameters,
		Headers:       r.Message.Headers,
		ContentType:   r.Message.ContentType,
		Content:       r.Message.Content,
		ContentLength: int64(len(r.Message.Content)),
	}

	queueMessages, err := s.ExchangeBO.PublishMessage(
		ctx,
		r.ExchangeCode,
		r.RoutingKeyOrPatternOrQueueCode,
		message,
		r.Vnamespace,
		cf,
		cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		return nil, err
	}

	return &pb.PublishMessageResponse{
		Message:       "Message published successfully",
		QueueMessages: queueMessages, // map[string]string where key=queueCode, value=messageID
	}, nil
}

func (s *ExchangeService) PublishStream(stream pb.ExchangeService_PublishStreamServer) error {
	ctx := stream.Context()
	tenantBO := bo.NewTenantBO(s.Config)
	tenantCache := make(map[string]*common.TenantContext)

	// gRPC streams are not safe for concurrent sends, use a dedicated channel and sender goroutine
	sendChan := make(chan *pb.PublishStreamResponse, s.Config.PublishBufferMaxSize*2)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case resp := <-sendChan:
				_ = stream.Send(resp)
			}
		}
	}()

	var recvCount int
	var startRecv time.Time
	
	for {
		req, err := stream.Recv()
		if err != nil {
			// Handle EOF or error
			return nil
		}

		if recvCount == 0 {
			startRecv = time.Now()
		}
		recvCount++
		if recvCount%100 == 0 {
			s.Config.Logger.Debug().Int("count", recvCount).Dur("duration", time.Since(startRecv)).Msg("Received 100 messages from stream.Recv()")
			startRecv = time.Now()
		}

		tenantCtx, ok := tenantCache[req.TenantCode]
		if !ok {
			tenant, _, _, err := tenantBO.GetTenant(ctx, req.TenantCode)
			if err != nil {
				s.Config.Logger.Error().Err(err).Str("tenantCode", req.TenantCode).Msg("Failed to get tenant in PublishStream")
				resp := &pb.PublishStreamResponse{
					ClientMessageId: req.ClientMessageId,
					Confirmed:       false,
					Error:           "Invalid tenant: " + err.Error(),
				}
				sendChan <- resp
				continue
			}

			cf := db.ColumnFamilyPrefix + strconv.Itoa(tenant.ColumnFamilyIndex)
			cfs := tenant.ID

			var tenantNode *dragonboat.RaftNode
			s.Config.TenantNodesLock.Lock()
			for i := range s.Config.TenantNodes {
				if s.Config.TenantNodes[i].ShardID == uint64(tenant.ShardId) {
					tenantNode = s.Config.TenantNodes[i]
					break
				}
			}
			s.Config.TenantNodesLock.Unlock()

			if tenantNode == nil {
				s.Config.Logger.Error().Str("tenantCode", req.TenantCode).Msg("No node found for tenant shard in PublishStream")
				resp := &pb.PublishStreamResponse{
					ClientMessageId: req.ClientMessageId,
					Confirmed:       false,
					Error:           "Tenant node not available",
				}
				sendChan <- resp
				continue
			}

			tenantCtx = &common.TenantContext{
				Tenant: &tenant,
				Node:   tenantNode,
				CF:     cf,
				CFS:    cfs,
			}
			tenantCache[req.TenantCode] = tenantCtx
		}

		// Convert protobuf message to models.QueueMessage
		message := models.QueueMessage{
			MessageID:     req.Message.MessageId,
			Handler:       req.Message.Handler,
			Priority:      int(req.Message.Priority),
			Parameters:    req.Message.Parameters,
			Headers:       req.Message.Headers,
			ContentType:   req.Message.ContentType,
			Content:       req.Message.Content,
			ContentLength: int64(len(req.Message.Content)),
		}

		bufferedMessage := buffer.PublishBufferedMessage{
			ClientMessageID: req.ClientMessageId,
			ExchangeCode:    req.ExchangeCode,
			RoutingKey:      req.RoutingKeyOrPatternOrQueueCode,
			VNamespace:      req.Vnamespace,
			Message:         message,
			CF:              tenantCtx.CF,
			CFS:             tenantCtx.CFS,
			Tenant:          tenantCtx.Tenant,
			TenantNode:      tenantCtx.Node,
			SendChan:        sendChan,
		}

		// Add to buffer (non-blocking unless flush happens)
		s.publishBuffer.Add(ctx, bufferedMessage)
	}
}
