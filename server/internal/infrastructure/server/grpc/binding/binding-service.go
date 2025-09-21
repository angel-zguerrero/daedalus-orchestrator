package binding

import (
	"context"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/server/common"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/binding"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
)

type BindingService struct {
	pb.UnimplementedBindingServiceServer
	startTime time.Time
	Config    *common.ServerConfing
	BindingBO *bo.BindingBO
}

func NewBindingService(config *common.ServerConfing) *BindingService {
	return &BindingService{
		Config:    config,
		BindingBO: bo.NewBindingBO(config),
	}
}

func (s *BindingService) CreateBinding(ctx context.Context, r *pb.CreateBindingRequest) (*pb.CreateBindingResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	// Set default target exchange type if not provided
	targetExchangeType := models.TargetExchangeTypeQueue
	if r.TargetExchangeType != "" {
		targetExchangeType = models.TargetExchangeType(r.TargetExchangeType)
	}

	// Set default binding type if not provided
	bindingType := models.BindingTypeClassic
	if r.BindingType != "" {
		bindingType = models.BindingType(r.BindingType)
	}

	// Set default XMatch if not provided
	xMatch := models.XMatchTypeAll
	if r.XMatch != "" {
		xMatch = models.XMatchType(r.XMatch)
	}

	binding, err := s.BindingBO.CreateBinding(
		ctx,
		r.Code,
		r.QueueCode,
		r.ExchangeCode,
		r.TargetExchangeCode,
		r.AlternateExchangeCode,
		r.Vnamespace,
		r.RoutingKey,
		r.Pattern,
		xMatch,
		bindingType,
		targetExchangeType,
		r.Headers,
		cf,
		cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		return nil, err
	}

	return &pb.CreateBindingResponse{
		Message: "Binding was created",
		Result: &pb.Binding{
			Id:                    binding.ID,
			Code:                  binding.Code,
			ExchangeCode:          r.ExchangeCode,
			QueueCode:             r.QueueCode,
			TargetExchangeCode:    r.TargetExchangeCode,
			AlternateExchangeCode: r.AlternateExchangeCode,
			Vnamespace:            binding.VNamespace,
			RoutingKey:            binding.RoutingKey,
			Pattern:               binding.Pattern,
			XMatch:                string(binding.XMatch),
			BindingType:           string(binding.BindingType),
			TargetExchangeType:    string(binding.TargetExchangeType),
			CreatedAt:             binding.CreatedAt.Format(time.RFC3339),
			UpdatedAt:             binding.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *BindingService) GetBinding(ctx context.Context, r *pb.GetBindingRequest) (*pb.GetBindingResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	binding, err := s.BindingBO.GetBinding(
		ctx,
		r.ExchangeCode,
		r.QueueCode,
		r.Vnamespace,
		cf,
		cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		return nil, err
	}

	return &pb.GetBindingResponse{
		Message: "Binding",
		Result: &pb.Binding{
			Id:                    binding.ID,
			Code:                  binding.Code,
			ExchangeCode:          binding.ExchangeCode,
			QueueCode:             binding.QueueCode,
			TargetExchangeCode:    binding.TargetExchangeCode,
			AlternateExchangeCode: binding.AlternateExchangeCode,
			Vnamespace:            binding.VNamespace,
			RoutingKey:            binding.RoutingKey,
			Pattern:               binding.Pattern,
			XMatch:                string(binding.XMatch),
			BindingType:           string(binding.BindingType),
			TargetExchangeType:    string(binding.TargetExchangeType),
			CreatedAt:             binding.CreatedAt.Format(time.RFC3339),
			UpdatedAt:             binding.UpdatedAt.Format(time.RFC3339),
			Headers:               binding.Headers,
		},
	}, nil
}

func (s *BindingService) GetBindings(ctx context.Context, r *pb.GetBindingsRequest) (*pb.GetBindingsResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	findResult, err := s.BindingBO.GetBindings(
		ctx,
		r.Q,
		r.Cursor,
		int(r.PageSize),
		r.Vnamespace,
		r.IncludeObjects,
		cf,
		cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		return nil, err
	}

	rBindings := []*pb.Binding{}

	// Use the simplified Binding model with virtual fields
	for _, e := range findResult.Entities {
		binding := &pb.Binding{
			Id:                    e.ID,
			Code:                  e.Code,
			ExchangeCode:          e.ExchangeCode,
			QueueCode:             e.QueueCode,
			TargetExchangeCode:    e.TargetExchangeCode,
			AlternateExchangeCode: e.AlternateExchangeCode,
			Vnamespace:            e.VNamespace,
			RoutingKey:            e.RoutingKey,
			Pattern:               e.Pattern,
			XMatch:                string(e.XMatch),
			BindingType:           string(e.BindingType),
			TargetExchangeType:    string(e.TargetExchangeType),
			CreatedAt:             e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:             e.UpdatedAt.Format(time.RFC3339),
		}

		// Add exchange if available and included
		if r.IncludeObjects && e.Exchange != nil {
			binding.Exchange = &pb.Exchange{
				Id:         e.Exchange.ID,
				Code:       e.Exchange.Code,
				Name:       e.Exchange.Name,
				Type:       string(e.Exchange.Type),
				Vnamespace: e.Exchange.VNamespace,
				CreatedAt:  e.Exchange.CreatedAt.Format(time.RFC3339),
				UpdatedAt:  e.Exchange.UpdatedAt.Format(time.RFC3339),
			}
		}

		// Add queue if available and included
		if r.IncludeObjects && e.Queue != nil {
			binding.Queue = &pb.Queue{
				Id:            e.Queue.ID,
				Code:          e.Queue.Code,
				Name:          e.Queue.Name,
				Vnamespace:    e.Queue.VNamespace,
				State:         string(e.Queue.State),
				Type:          string(e.Queue.Type),
				MessagesCount: int32(e.Queue.MessagesCount),
				CreatedAt:     e.Queue.CreatedAt.Format(time.RFC3339),
				UpdatedAt:     e.Queue.UpdatedAt.Format(time.RFC3339),
			}
		}

		// Add target exchange if available and included
		if r.IncludeObjects && e.TargetExchange != nil {
			binding.TargetExchange = &pb.Exchange{
				Id:         e.TargetExchange.ID,
				Code:       e.TargetExchange.Code,
				Name:       e.TargetExchange.Name,
				Type:       string(e.TargetExchange.Type),
				Vnamespace: e.TargetExchange.VNamespace,
				CreatedAt:  e.TargetExchange.CreatedAt.Format(time.RFC3339),
				UpdatedAt:  e.TargetExchange.UpdatedAt.Format(time.RFC3339),
			}
		}

		// Add alternate exchange if available and included
		if r.IncludeObjects && e.AlternateExchange != nil {
			binding.AlternateExchange = &pb.Exchange{
				Id:         e.AlternateExchange.ID,
				Code:       e.AlternateExchange.Code,
				Name:       e.AlternateExchange.Name,
				Type:       string(e.AlternateExchange.Type),
				Vnamespace: e.AlternateExchange.VNamespace,
				CreatedAt:  e.AlternateExchange.CreatedAt.Format(time.RFC3339),
				UpdatedAt:  e.AlternateExchange.UpdatedAt.Format(time.RFC3339),
			}
		}

		// Add headers if available and included
		if r.IncludeObjects && e.Headers != nil {
			binding.Headers = e.Headers
		}

		rBindings = append(rBindings, binding)
	}

	return &pb.GetBindingsResponse{
		Message: "Binding list",
		Result: &pb.BindingFindResult{
			Entities: rBindings,
			Cursor:   findResult.Cursor,
		},
	}, nil
}

func (s *BindingService) DeleteBinding(ctx context.Context, r *pb.DeleteBindingRequest) (*pb.DeleteBindingResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	err := s.BindingBO.DeleteBinding(
		ctx,
		r.Code,
		r.Vnamespace,
		cf,
		cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteBindingResponse{
		Message: fmt.Sprintf("Binding with code %s in namespace %s was deleted", r.Code, r.Vnamespace),
	}, nil
}
