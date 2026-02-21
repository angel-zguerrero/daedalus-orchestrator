package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	job_worker_command "deadalus-orch/server/internal/usecase/command/job-worker"
	"deadalus-orch/shared/models"
	"errors"
	"fmt"
)

type JobWorkerBO struct {
	Config *common.ServerConfing
}

func NewJobWorkerBO(Config *common.ServerConfing) *JobWorkerBO {
	return &JobWorkerBO{
		Config: Config,
	}
}

func (bo *JobWorkerBO) ClaimWork(ctx context.Context, workerId string, workerName string, Information map[string]string, ClaimWorkCapacityPolicies map[string]models.ClaimWorkCapacityPolicy) ([]models.QueueMessage, error) {
	// Upsert the JobWorker: update LastHeartbeat and TTL on every ClaimWork call
	upsertCmd := &job_worker_command.UpsertJobWorkerCommand{
		JobWorkers: []models.JobWorker{
			{
				ID:                        workerId,
				Name:                      workerName,
				Information:               Information,
				ClaimWorkCapacityPolicies: ClaimWorkCapacityPolicies,
			},
		},
	}

	_, err := dragonboat.ExecuteRepositoryCommand[[]models.JobWorker](
		bo.Config.MasterNode,
		ctx,
		upsertCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"upsert job worker",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert job worker: %w", err)
	}

	return nil, nil
}

func (bo *JobWorkerBO) GetJobWorker(ctx context.Context, jobWorkerID string) (models.JobWorker, error) {
	findJobWorkerCommand := &job_worker_command.PaginateJobWorkersCommand{
		Q:        jobWorkerID,
		PageSize: 1,
	}

	// Actually, should use a FindJobWorkerCommand if it exists, or Paginate with Q=ID
	// For now, let's use PaginateJobWorkersCommand as a general way to get them.
	// RE-PLAN: I'll use PaginateJobWorkersCommand for GetJobWorkers and maybe create a FindJobWorkerCommand later if needed.
	// But let's check if I can just use Paginate for both.

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.JobWorker]](
		bo.Config.MasterNode,
		ctx,
		findJobWorkerCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"find jobWorker",
	)

	if err != nil {
		return models.JobWorker{}, err
	}

	if len(findResult.Entities) == 0 {
		return models.JobWorker{}, errors.New("JobWorker not found")
	}

	return findResult.Entities[0], nil
}

func (bo *JobWorkerBO) GetJobWorkers(ctx context.Context, q string, cursor string, pageSize int) (db.FindResult[models.JobWorker], error) {
	paginateJobWorkersCommand := &job_worker_command.PaginateJobWorkersCommand{
		Cursor:   cursor,
		PageSize: pageSize,
		Q:        q,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.JobWorker]](
		bo.Config.MasterNode,
		ctx,
		paginateJobWorkersCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"paginate jobWorkers",
	)
	if err != nil {
		return db.FindResult[models.JobWorker]{}, fmt.Errorf("paginate jobWorkers failed: %w", err)
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.JobWorker{}
	}

	return findResult, nil
}
