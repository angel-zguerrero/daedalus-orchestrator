package scheduledjob

import (
	"context"
	"fmt"
	"time"

	"deadalus-orch/server/internal/infrastructure/server/common"
	pb "deadalus-orch/server/internal/infrastructure/server/grpc/proto/pb/scheduledjob"
	bo "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
)

type ScheduledJobService struct {
	pb.UnimplementedScheduledJobServiceServer
	Config         *common.ServerConfing
	ScheduledJobBO *bo.ScheduledJobBO
}

func NewScheduledJobService(config *common.ServerConfing) *ScheduledJobService {
	return &ScheduledJobService{
		Config:         config,
		ScheduledJobBO: bo.NewScheduledJobBO(config),
	}
}

func ConvertScheduledJobToProto(m *models.ScheduledJob) *pb.ScheduledJob {
	if m == nil {
		return nil
	}
	runAtStr := ""
	if m.RunAt != nil && !m.RunAt.IsZero() {
		runAtStr = m.RunAt.Format(time.RFC3339)
	}
	return &pb.ScheduledJob{
		Id:                             m.ID,
		Code:                           m.Code,
		TenantId:                       m.TenantID,
		TargetType:                     m.TargetType,
		TargetId:                       m.TargetID,
		TargetCode:                     m.TargetCode,
		RoutingKeyOrPatternOrQueueCode: m.RoutingKeyOrPatternOrQueueCode,
		Vnamespace:                     m.VNamespace,
		Content:                        m.Content,
		ContentType:                    m.ContentType,
		Headers:                        m.Headers,
		Handler:                        m.Handler,
		Parameters:                     m.Parameters,
		Priority:                       int32(m.Priority),
		State:                          string(m.State),
		Type:                           string(m.Type),
		Every:                          m.Every,
		CronExpression:                 m.CronExpression,
		RunAt:                          runAtStr,
		RunAfter:                       m.RunAfter,
		NextRunAt:                      m.NextRunAt.Format(time.RFC3339),
		CreatedAt:                      m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                      m.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *ScheduledJobService) CreateOneOffScheduledJob(
	ctx context.Context,
	r *pb.CreateOneOffScheduledJobRequest,
) (*pb.CreateScheduledJobResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	var runAt *time.Time
	if r.RunAt != "" {
		parsedRunAt, err := time.Parse(time.RFC3339, r.RunAt)
		if err != nil {
			return nil, fmt.Errorf("invalid runAt timestamp format (expected RFC3339): %w", err)
		}
		runAt = &parsedRunAt
	}

	job, err := s.ScheduledJobBO.CreateOneOffScheduledJob(
		ctx,
		r.Code,
		r.TenantCode,
		r.TargetType,
		r.TargetCode,
		r.Vnamespace,
		r.Content,
		r.ContentType,
		r.Headers,
		r.Handler,
		r.Parameters,
		int(r.Priority),
		runAt,
		r.RunAfter,
		cf, cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		return nil, err
	}

	return &pb.CreateScheduledJobResponse{
		Message: "OneOff scheduled job created successfully",
		Result:  ConvertScheduledJobToProto(&job),
	}, nil
}

func (s *ScheduledJobService) CreateRecurringScheduledJob(
	ctx context.Context,
	r *pb.CreateRecurringScheduledJobRequest,
) (*pb.CreateScheduledJobResponse, error) {
	tenant, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	job, err := s.ScheduledJobBO.CreateRecurringScheduledJob(
		ctx,
		r.Code,
		r.TenantCode,
		r.TargetType,
		r.TargetCode,
		r.Vnamespace,
		r.Content,
		r.ContentType,
		r.Headers,
		r.Handler,
		r.Parameters,
		int(r.Priority),
		r.Every,
		r.CronExpression,
		cf, cfs,
		tenant,
		tenantNode,
	)
	if err != nil {
		return nil, err
	}

	return &pb.CreateScheduledJobResponse{
		Message: "Recurring scheduled job created successfully",
		Result:  ConvertScheduledJobToProto(&job),
	}, nil
}

func (s *ScheduledJobService) GetScheduledJob(
	ctx context.Context,
	r *pb.GetScheduledJobRequest,
) (*pb.GetScheduledJobResponse, error) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	job, err := s.ScheduledJobBO.GetScheduledJob(ctx, r.Id, cf, cfs, tenantNode)
	if err != nil {
		return nil, err
	}

	return &pb.GetScheduledJobResponse{
		Message: "Scheduled job found",
		Result:  ConvertScheduledJobToProto(&job),
	}, nil
}

func (s *ScheduledJobService) ListScheduledJobs(
	ctx context.Context,
	r *pb.ListScheduledJobsRequest,
) (*pb.ListScheduledJobsResponse, error) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	res, err := s.ScheduledJobBO.ListScheduledJobs(ctx, r.Cursor, int(r.PageSize), r.Vnamespace, cf, cfs, tenantNode)
	if err != nil {
		return nil, err
	}

	pEntities := make([]*pb.ScheduledJob, len(res.Entities))
	for i := range res.Entities {
		pEntities[i] = ConvertScheduledJobToProto(&res.Entities[i])
	}

	return &pb.ListScheduledJobsResponse{
		Message: "Scheduled job list",
		Result: &pb.ScheduledJobFindResult{
			Entities: pEntities,
			Cursor:   res.Cursor,
		},
	}, nil
}

func (s *ScheduledJobService) DeleteScheduledJob(
	ctx context.Context,
	r *pb.DeleteScheduledJobRequest,
) (*pb.DeleteScheduledJobResponse, error) {
	_, tenantNode, cf, cfs := common.MustGetTenantData(ctx)

	err := s.ScheduledJobBO.DeleteScheduledJob(ctx, r.Id, cf, cfs, tenantNode)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteScheduledJobResponse{
		Message: fmt.Sprintf("Scheduled job %s deleted", r.Id),
	}, nil
}
