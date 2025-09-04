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
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantId)
	if err != nil {
		return nil, err
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
		r.QueueCode,
		r.ExchangeCode,
		r.Vnamespace,
		r.RoutingKey,
		r.Pattern,
		xMatch,
		bindingType,
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
			Id:           binding.ID,
			ExchangeCode: r.ExchangeCode,
			QueueCode:    r.QueueCode,
			Vnamespace:   binding.VNamespace,
			RoutingKey:   binding.RoutingKey,
			Pattern:      binding.Pattern,
			XMatch:       string(binding.XMatch),
			BindingType:  string(binding.BindingType),
			CreatedAt:    binding.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    binding.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *BindingService) GetBinding(ctx context.Context, r *pb.GetBindingRequest) (*pb.GetBindingResponse, error) {
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantId)
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
			Id:           binding.ID,
			ExchangeCode: r.ExchangeCode,
			QueueCode:    r.QueueCode,
			Vnamespace:   binding.VNamespace,
			RoutingKey:   binding.RoutingKey,
			Pattern:      binding.Pattern,
			XMatch:       string(binding.XMatch),
			BindingType:  string(binding.BindingType),
			CreatedAt:    binding.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    binding.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *BindingService) GetBindings(ctx context.Context, r *pb.GetBindingsRequest) (*pb.GetBindingsResponse, error) {
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantId)
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

	if r.IncludeObjects {
		// Cast to BindingWithObjects result
		if bindingsWithObjects, ok := findResult.(db.FindResult[models.BindingWithObjects]); ok {
			for _, e := range bindingsWithObjects.Entities {
				binding := &pb.Binding{
					Id:           e.ID,
					ExchangeCode: e.ExchangeCode,
					QueueCode:    e.QueueCode,
					Vnamespace:   e.VNamespace,
					RoutingKey:   e.RoutingKey,
					Pattern:      e.Pattern,
					XMatch:       string(e.XMatch),
					BindingType:  string(e.BindingType),
					CreatedAt:    e.CreatedAt.Format(time.RFC3339),
					UpdatedAt:    e.UpdatedAt.Format(time.RFC3339),
				}

				// Add exchange if available
				if e.Exchange != nil {
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

				// Add queue if available
				if e.Queue != nil {
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

				rBindings = append(rBindings, binding)
			}

			return &pb.GetBindingsResponse{
				Message: "Binding list",
				Result: &pb.BindingFindResult{
					Entities: rBindings,
					Cursor:   bindingsWithObjects.Cursor,
				},
			}, nil
		}
	} else {
		// Cast to regular Binding result
		if bindings, ok := findResult.(db.FindResult[models.Binding]); ok {
			for _, e := range bindings.Entities {
				binding := &pb.Binding{
					Id:           e.ID,
					ExchangeCode: "", // Will need to resolve from ExchangeID if needed
					QueueCode:    "", // Will need to resolve from QueueID if needed
					Vnamespace:   e.VNamespace,
					RoutingKey:   e.RoutingKey,
					Pattern:      e.Pattern,
					XMatch:       string(e.XMatch),
					BindingType:  string(e.BindingType),
					CreatedAt:    e.CreatedAt.Format(time.RFC3339),
					UpdatedAt:    e.UpdatedAt.Format(time.RFC3339),
				}
				rBindings = append(rBindings, binding)
			}

			return &pb.GetBindingsResponse{
				Message: "Binding list",
				Result: &pb.BindingFindResult{
					Entities: rBindings,
					Cursor:   bindings.Cursor,
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("unexpected result type")
}

func (s *BindingService) DeleteBinding(ctx context.Context, r *pb.DeleteBindingRequest) (*pb.DeleteBindingResponse, error) {
	tenant, _, _, err := s.TenantBO.GetTenant(ctx, r.TenantId)
	if err != nil {
		return nil, err
	}

	err = s.BindingBO.DeleteBinding(
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

	return &pb.DeleteBindingResponse{
		Message: fmt.Sprintf("Binding between exchange %s and queue %s in namespace %s was deleted", r.ExchangeCode, r.QueueCode, r.Vnamespace),
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
