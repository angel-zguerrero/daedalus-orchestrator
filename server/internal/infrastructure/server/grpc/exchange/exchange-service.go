package exchange

import (
	"context"
	"strconv"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/exchange"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
)

type ExchangeService struct {
	pb.UnimplementedExchangeServiceServer
	startTime  time.Time
	Config     *common.ServerConfing
	ExchangeBO *bo.ExchangeBO
	TenantBO   *bo.TenantBO
}

func NewExchangeService(config *common.ServerConfing) *ExchangeService {
	return &ExchangeService{
		startTime:  time.Now(),
		Config:     config,
		ExchangeBO: bo.NewExchangeBO(config),
		TenantBO:   bo.NewTenantBO(config),
	}
}

func (s *ExchangeService) CreateExchange(ctx context.Context, r *pb.CreateExchangeRequest) (*pb.CreateExchangeResponse, error) {
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantCode)
	if err != nil {
		return nil, err
	}

	exchange, err := s.ExchangeBO.CreateExchange(ctx, r.Code, r.Vnamespace, r.Name, models.ExchangeType(r.Type), r.Headers, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
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
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantCode)
	if err != nil {
		return nil, err
	}

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

	exchangesResult, err := s.ExchangeBO.BulkCreateExchange(ctx, exchanges, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
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
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantCode)
	if err != nil {
		return nil, err
	}

	exchange, err := s.ExchangeBO.GetExchange(ctx, r.Code, r.Vnamespace, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
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
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantCode)
	if err != nil {
		return nil, err
	}

	page := int(r.PageSize)
	if page < 2 {
		page = 50
	} else if page > 1000 {
		page = 1000
	}

	findResult, err := s.ExchangeBO.GetExchanges(ctx, r.Q, r.Cursor, page, r.Vnamespace, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
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
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantCode)
	if err != nil {
		return nil, err
	}

	err = s.ExchangeBO.DeleteExchange(ctx, r.Code, r.Vnamespace, db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex), tenant.ID)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteExchangeResponse{
		Message: "Exchange " + r.Code + " in namespace " + r.Vnamespace + " was deleted",
	}, nil
}

func (s *ExchangeService) PublishMessage(ctx context.Context, r *pb.PublishMessageRequest) (*pb.PublishMessageResponse, error) {
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantCode)
	if err != nil {
		return nil, err
	}

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

	queueCodes, err := s.ExchangeBO.PublishMessage(
		ctx,
		r.ExchangeCode,
		r.RoutingKeyOrPatternOrQueueCode,
		r.Vnamespace,
		message,
		db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex),
		tenant.ID,
	)
	if err != nil {
		return nil, err
	}

	return &pb.PublishMessageResponse{
		Message:    "Message published successfully",
		QueueCodes: queueCodes,
	}, nil
}
