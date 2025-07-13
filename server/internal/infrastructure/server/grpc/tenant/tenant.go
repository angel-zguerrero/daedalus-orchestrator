package tenant

import (
	"context"
	"time"

	"deadalus-orch/server/internal/infrastructure/server/common"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/tenant"
	bo "deadalus-orch/server/internal/usecase/business-logic"
)

type TenantService struct {
	pb.UnimplementedTenantServiceServer
	startTime time.Time
	Config    *common.ServerConfing
	TenantBO  *bo.TenantBO
}

func NewTenantService(config *common.ServerConfing) *TenantService {
	return &TenantService{
		startTime: time.Now(),
		Config:    config,
		TenantBO:  bo.NewTenantBO(config),
	}
}

func (s *TenantService) AssertTenant(ctx context.Context, r *pb.AssertTenantRequest) (*pb.AssertTenantResponse, error) {
	tenantInMaster, err := s.TenantBO.CreateTenant(ctx, r.Code, r.Name)
	if err != nil {
		return nil, err
	}

	return &pb.AssertTenantResponse{
		Message: "Tenant was created",
		Result: &pb.Tenant{
			ID:        tenantInMaster.ID,
			Name:      tenantInMaster.Name,
			ShardId:   int64(tenantInMaster.ShardId),
			Code:      tenantInMaster.Code,
			Status:    string(tenantInMaster.Status),
			CreatedAt: tenantInMaster.CreatedAt.Format(time.RFC3339),
			UpdatedAt: tenantInMaster.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (s *TenantService) GetTenantInfo(ctx context.Context, r *pb.TenantInfoRequest) (*pb.TenantInfoResponse, error) {
	tenantInMaster, node, nodeHostInfo, err := s.TenantBO.GetTenant(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	response := &pb.TenantInfoResponse{
		Message: "Tenant",
		Result: &pb.Tenant{
			ID:        tenantInMaster.ID,
			Name:      tenantInMaster.Name,
			Code:      tenantInMaster.Code,
			ShardId:   int64(tenantInMaster.ShardId),
			Status:    string(tenantInMaster.Status),
			CreatedAt: tenantInMaster.CreatedAt.Format(time.RFC3339),
			UpdatedAt: tenantInMaster.UpdatedAt.Format(time.RFC3339),
		},
	}

	if nodeHostInfo != nil {
		response.Node = &pb.Node{
			SelfMember: node.SelfMember,
			ShardID:    node.ShardID,
			Roles:      node.Roles,
			NodeHostInfo: &pb.NodeHostInfo{
				RaftAddress:    nodeHostInfo.RaftAddress,
				ServiceAddress: nodeHostInfo.ServiceAddress,
			},
		}
	}

	return response, nil
}

func (s *TenantService) DeleteTenant(ctx context.Context, r *pb.DeleteTenantRequest) (*pb.DeleteTenantResponse, error) {
	err := s.TenantBO.DeleteTenant(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteTenantResponse{Message: "Tenant " + r.ID + " was deleted"}, nil
}

func (s *TenantService) GetTenants(ctx context.Context, r *pb.GetTenantsRequest) (*pb.GetTenantsResponse, error) {
	findResult, err := s.TenantBO.GetTenants(ctx, r.Cursor, int(r.PageSize))
	if err != nil {
		return nil, err
	}

	tenants := make([]*pb.Tenant, len(findResult.Entities))
	for i, t := range findResult.Entities {
		tenants[i] = &pb.Tenant{
			ID:        t.ID,
			Name:      t.Name,
			Code:      t.Code,
			ShardId:   int64(t.ShardId),
			Status:    string(t.Status),
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
			UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
		}
	}

	return &pb.GetTenantsResponse{
		Message: "Tenant list",
		Result: &pb.FindResult{
			Tenants:    tenants,
			NextCursor: findResult.Cursor,
		},
	}, nil
}
