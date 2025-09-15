package binding

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
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
	TenantBO  *bo.TenantBO
}

func NewBindingService(config *common.ServerConfing) *BindingService {
	return &BindingService{
		Config:    config,
		BindingBO: bo.NewBindingBO(config),
		TenantBO:  bo.NewTenantBO(config),
	}
}

func (s *BindingService) CreateBinding(ctx context.Context, r *pb.CreateBindingRequest) (*pb.CreateBindingResponse, error) {
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantCode)
	if err != nil {
		return nil, err
	}

	// Validate that Code is provided
	if r.Code == "" {
		return nil, fmt.Errorf("code is required")
	}

	// Set default target exchange type if not provided
	targetExchangeType := models.TargetExchangeTypeQueue
	if r.TargetExchangeType != "" {
		if !isValidTargetExchangeType(r.TargetExchangeType) {
			return nil, fmt.Errorf("invalid target exchange type: %s. Valid types are: queue, exchange", r.TargetExchangeType)
		}
		targetExchangeType = models.TargetExchangeType(r.TargetExchangeType)
	}

	// Validate target exchange type specific requirements
	if targetExchangeType == models.TargetExchangeTypeQueue {
		if r.BindingType == "classic" && r.QueueCode == "" {
			return nil, fmt.Errorf("queueCode is required for classic bindings when targetExchangeType is queue")
		}
		if r.TargetExchangeCode != "" {
			return nil, fmt.Errorf("targetExchangeCode should not be specified when targetExchangeType is queue")
		}
	} else if targetExchangeType == models.TargetExchangeTypeExchange {
		if r.TargetExchangeCode == "" {
			return nil, fmt.Errorf("targetExchangeCode is required when targetExchangeType is exchange")
		}
		if r.QueueCode != "" {
			return nil, fmt.Errorf("queueCode should not be specified when targetExchangeType is exchange")
		}
		if r.BindingType == "dynamic" {
			return nil, fmt.Errorf("exchange targets are not supported for dynamic bindings")
		}
	}

	// Legacy validation for backward compatibility
	if r.BindingType == "classic" && targetExchangeType == models.TargetExchangeTypeQueue && r.QueueCode == "" {
		return nil, fmt.Errorf("queueCode is required for classic bindings")
	}
	if r.BindingType == "dynamic" && r.QueueCode != "" {
		return nil, fmt.Errorf("queueCode should not be specified for dynamic bindings")
	}

	// Validate binding type
	if r.BindingType != "" && !isValidBindingType(r.BindingType) {
		return nil, fmt.Errorf("invalid binding type: %s. Valid types are: classic, dynamic", r.BindingType)
	}

	// Validate XMatch type
	if r.XMatch != "" && !isValidXMatchType(r.XMatch) {
		return nil, fmt.Errorf("invalid xMatch type: %s. Valid types are: all, any", r.XMatch)
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
		db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex),
		tenant.ID,
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
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantCode)
	if err != nil {
		return nil, err
	}

	binding, err := s.BindingBO.GetBinding(
		ctx,
		r.ExchangeCode,
		r.QueueCode,
		r.Vnamespace,
		db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex),
		tenant.ID,
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
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantCode)
	if err != nil {
		return nil, err
	}

	findResult, err := s.BindingBO.GetBindings(
		ctx,
		r.Q,
		r.Cursor,
		int(r.PageSize),
		r.Vnamespace,
		r.IncludeObjects,
		db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex),
		tenant.ID,
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
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantCode)
	if err != nil {
		return nil, err
	}

	err = s.BindingBO.DeleteBinding(
		ctx,
		r.Code,
		r.Vnamespace,
		db.ColumnFamilyPrefix+strconv.Itoa(tenant.ColumnFamilyIndex),
		tenant.ID,
	)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteBindingResponse{
		Message: fmt.Sprintf("Binding with code %s in namespace %s was deleted", r.Code, r.Vnamespace),
	}, nil
}

// isValidBindingType validates if the binding type is one of the allowed types
func isValidBindingType(bindingType string) bool {
	switch bindingType {
	case "classic", "dynamic":
		return true
	default:
		return false
	}
}

// isValidXMatchType validates if the XMatch type is one of the allowed types
func isValidXMatchType(xMatch string) bool {
	switch xMatch {
	case "all", "any":
		return true
	default:
		return false
	}
}

// isValidTargetExchangeType validates if the target exchange type is one of the allowed types
func isValidTargetExchangeType(targetExchangeType string) bool {
	switch targetExchangeType {
	case "queue", "exchange":
		return true
	default:
		return false
	}
}
