package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	scheduled_job_command "deadalus-orch/server/internal/usecase/command/scheduled-job"
	"deadalus-orch/shared/models"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ScheduledJobBO struct {
	Config     *common.ServerConfing
	QueueBO    *QueueBO
	ExchangeBO *ExchangeBO
}

func NewScheduledJobBO(config *common.ServerConfing) *ScheduledJobBO {
	return &ScheduledJobBO{
		Config:     config,
		QueueBO:    NewQueueBO(config),
		ExchangeBO: NewExchangeBO(config),
	}
}

func (bo *ScheduledJobBO) resolveTarget(
	ctx context.Context,
	targetType string,
	targetCode string,
	vnamespace string,
	cf, cfs string,
	tenant *models.TenantInMaster,
	tenantNode *dragonboat.RaftNode,
) (targetID string, resolvedTargetCode string, routingKey string, err error) {
	if targetType == string(models.ScheduledJobTargetQueue) {
		q, err := bo.QueueBO.GetQueue(ctx, targetCode, vnamespace, false, cf, cfs, tenant, tenantNode)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to resolve queue target %s: %w", targetCode, err)
		}
		return q.ID, q.Code, q.Code, nil
	}

	if targetType == string(models.ScheduledJobTargetExchange) {
		ex, err := bo.ExchangeBO.GetExchange(ctx, targetCode, vnamespace, cf, cfs, tenant, tenantNode)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to resolve exchange target %s: %w", targetCode, err)
		}
		return ex.ID, ex.Code, targetCode, nil
	}

	return "", "", "", fmt.Errorf("invalid targetType: %s. Must be 'queue' or 'exchange'", targetType)
}

func (bo *ScheduledJobBO) CreateOneOffScheduledJob(
	ctx context.Context,
	tenantCode string,
	targetType string,
	targetCode string,
	vnamespace string,
	content string,
	contentType string,
	headers map[string]string,
	handler string,
	parameters map[string]string,
	priority int,
	runAt *time.Time,
	runAfter string,
	cf, cfs string,
	tenant *models.TenantInMaster,
	tenantNode *dragonboat.RaftNode,
) (models.ScheduledJob, error) {
	if tenant.Status == models.PendingForDeletion {
		return models.ScheduledJob{}, errors.New("cannot create scheduled job when tenant is pending for deletion")
	}

	targetID, resTargetCode, routingKey, err := bo.resolveTarget(ctx, targetType, targetCode, vnamespace, cf, cfs, tenant, tenantNode)
	if err != nil {
		return models.ScheduledJob{}, err
	}

	now := time.Now().UTC()
	nextRunAt, err := utils.CalculateNextRunAt(
		models.ScheduledJobOneOff,
		runAt,
		runAfter,
		"",
		"",
		now,
	)
	if err != nil {
		return models.ScheduledJob{}, err
	}

	job := models.ScheduledJob{
		ID:                             uuid.New().String(),
		TenantID:                       tenant.ID,
		TargetType:                     targetType,
		TargetID:                       targetID,
		TargetCode:                     resTargetCode,
		RoutingKeyOrPatternOrQueueCode: routingKey,
		VNamespace:                     vnamespace,
		Content:                        content,
		ContentType:                    contentType,
		Headers:                        headers,
		Handler:                        handler,
		Parameters:                     parameters,
		Priority:                       priority,
		State:                          models.ScheduledJobIdle,
		Type:                           models.ScheduledJobOneOff,
		RunAt:                          runAt,
		RunAfter:                       runAfter,
		NextRunAt:                      nextRunAt,
		CreatedAt:                      now,
		UpdatedAt:                      now,
	}

	cmd := &scheduled_job_command.CreateScheduledJobCommand{
		ScheduledJob: job,
		CF:           cf,
		CFS:          cfs,
	}

	created, err := dragonboat.ExecuteRepositoryCommand[models.ScheduledJob](
		tenantNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"create one-off scheduled job",
	)
	if err != nil {
		return models.ScheduledJob{}, err
	}

	return created, nil
}

func (bo *ScheduledJobBO) CreateRecurringScheduledJob(
	ctx context.Context,
	tenantCode string,
	targetType string,
	targetCode string,
	vnamespace string,
	content string,
	contentType string,
	headers map[string]string,
	handler string,
	parameters map[string]string,
	priority int,
	every string,
	cronExpr string,
	cf, cfs string,
	tenant *models.TenantInMaster,
	tenantNode *dragonboat.RaftNode,
) (models.ScheduledJob, error) {
	if tenant.Status == models.PendingForDeletion {
		return models.ScheduledJob{}, errors.New("cannot create scheduled job when tenant is pending for deletion")
	}

	targetID, resTargetCode, routingKey, err := bo.resolveTarget(ctx, targetType, targetCode, vnamespace, cf, cfs, tenant, tenantNode)
	if err != nil {
		return models.ScheduledJob{}, err
	}

	now := time.Now().UTC()
	nextRunAt, err := utils.CalculateNextRunAt(
		models.ScheduledJobRecurring,
		nil,
		"",
		every,
		cronExpr,
		now,
	)
	if err != nil {
		return models.ScheduledJob{}, err
	}

	job := models.ScheduledJob{
		ID:                             uuid.New().String(),
		TenantID:                       tenant.ID,
		TargetType:                     targetType,
		TargetID:                       targetID,
		TargetCode:                     resTargetCode,
		RoutingKeyOrPatternOrQueueCode: routingKey,
		VNamespace:                     vnamespace,
		Content:                        content,
		ContentType:                    contentType,
		Headers:                        headers,
		Handler:                        handler,
		Parameters:                     parameters,
		Priority:                       priority,
		State:                          models.ScheduledJobIdle,
		Type:                           models.ScheduledJobRecurring,
		Every:                          every,
		CronExpression:                 cronExpr,
		NextRunAt:                      nextRunAt,
		CreatedAt:                      now,
		UpdatedAt:                      now,
	}

	cmd := &scheduled_job_command.CreateScheduledJobCommand{
		ScheduledJob: job,
		CF:           cf,
		CFS:          cfs,
	}

	created, err := dragonboat.ExecuteRepositoryCommand[models.ScheduledJob](
		tenantNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"create recurring scheduled job",
	)
	if err != nil {
		return models.ScheduledJob{}, err
	}

	return created, nil
}

func (bo *ScheduledJobBO) GetScheduledJob(
	ctx context.Context,
	id string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (models.ScheduledJob, error) {
	cmd := &scheduled_job_command.GetScheduledJobCommand{
		ID:  id,
		CF:  cf,
		CFS: cfs,
	}

	job, err := dragonboat.ExecuteRepositoryQuery[models.ScheduledJob](
		tenantNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"get scheduled job",
	)
	if err != nil {
		return models.ScheduledJob{}, err
	}

	return job, nil
}

func (bo *ScheduledJobBO) ListScheduledJobs(
	ctx context.Context,
	cursor string,
	pageSize int,
	vnamespace string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) (db.FindResult[models.ScheduledJob], error) {
	cmd := &scheduled_job_command.PaginateScheduledJobsCommand{
		PageSize:   pageSize,
		Cursor:     cursor,
		VNamespace: vnamespace,
		CF:         cf,
		CFS:        cfs,
	}

	res, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.ScheduledJob]](
		tenantNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"list scheduled jobs",
	)
	if err != nil {
		return db.FindResult[models.ScheduledJob]{}, err
	}

	if res.Entities == nil {
		res.Entities = []models.ScheduledJob{}
	}

	return res, nil
}

func (bo *ScheduledJobBO) DeleteScheduledJob(
	ctx context.Context,
	id string,
	cf, cfs string,
	tenantNode *dragonboat.RaftNode,
) error {
	cmd := &scheduled_job_command.DeleteScheduledJobCommand{
		ID:  id,
		CF:  cf,
		CFS: cfs,
	}

	_, err := dragonboat.ExecuteRepositoryCommand[bool](
		tenantNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"delete scheduled job",
	)
	return err
}
