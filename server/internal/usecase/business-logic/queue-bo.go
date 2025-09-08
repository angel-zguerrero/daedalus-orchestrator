package business_logic

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"fmt"

	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type QueueBO struct {
	Config *common.ServerConfing
}

func NewQueueBO(Config *common.ServerConfing) *QueueBO {
	return &QueueBO{
		Config: Config,
	}
}

func (bo *QueueBO) CreateQueue(ctx context.Context, code, vnamespace, name string, queueType models.QueueType, headers map[string]string, cf, cfs string) (models.Queue, error) {
	queue := &models.Queue{
		ID:              strings.ReplaceAll(uuid.New().String(), "-", ""),
		Code:            code,
		Name:            name,
		Type:            queueType,
		VNamespace:      vnamespace,
		State:           models.QueueActive, // Default state
		TTLQueue:        0,                  // Default TTL
		AllowDuplicated: true,               // Default allow duplicated
		MaxAttempts:     1,                  // Default max attempts
		Headers:         headers,            // Add headers support
	}

	createdList, err := bo.BulkCreateQueue(ctx, []*models.Queue{queue}, cf, cfs)
	if err != nil {
		return models.Queue{}, err
	}
	return createdList[0], nil
}

func (bo *QueueBO) BulkCreateQueue(ctx context.Context, queues []*models.Queue, cf, cfs string) ([]models.Queue, error) {
	if len(queues) == 0 {
		return nil, errors.New("no queues provided")
	}

	// Asegurar IDs válidos
	for _, t := range queues {
		if t.ID == "" {
			t.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
		}
		// Set default state if not provided
		if t.State == "" {
			t.State = models.QueueActive
		}
		// Set default values for new properties if not provided
		if t.MaxAttempts == 0 {
			t.MaxAttempts = 1
		}
		// TTLQueue defaults to 0, which is valid
		// AllowDuplicated defaults to false (Go bool default), but we want true
		// Note: In bulk creation, the caller should set these values explicitly
	}

	assertQueueCommand := &queue_command.AssertQueueCommand{
		Queues: make([]models.Queue, len(queues)),
		CF:     cf,
		CFS:    cfs,
	}
	for i, t := range queues {
		assertQueueCommand.Queues[i] = *t
	}

	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout*time.Duration(len(queues)))
	defer writeCancel()

	fsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  assertQueueCommand,
	}

	result, err := bo.Config.TenantNodesDictionary[cfs].Write(writeCtx, fsmCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Failed to assert queues (bulk)")
		return nil, fmt.Errorf("failed to assert queues (bulk): %w", err)
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Bulk queue creation command returned unexpected result type")
		return nil, fmt.Errorf("bulk queue creation command returned decode error: %w", err)
	}

	if parsedResult.Error != "" {
		return nil, fmt.Errorf("bulk queue creation failed: %s", parsedResult.Error)
	}

	created := parsedResult.Result.([]models.Queue)

	return created, nil
}

func (bo *QueueBO) GetQueue(ctx context.Context, queueCode, vnamespace string, includeHeaders bool, cf, cfs string) (models.Queue, error) {
	findQueueCommand := &queue_command.FindQueueCommand{
		Code:           queueCode,
		VNamespace:     vnamespace,
		IncludeHeaders: includeHeaders,
		CF:             cf,
		CFS:            cfs,
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: findQueueCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.TenantNodesDictionary[cfs].Read(readCtx, *queryCommand)
	if err != nil {
		if strings.Contains(err.Error(), "cannot encode nil pointer of type") {
			return models.Queue{}, errors.New("Queue not found")
		}
		bo.Config.Logger.Error().Err(err).Msg("Find queue command failed")
		return models.Queue{}, errors.New("Find queue command failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Find queue command failed")
		return models.Queue{}, errors.New("Find queue command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find queue command failed")
		return models.Queue{}, errors.New("Find queue command failed")
	}

	if parsedResult.Result == nil {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find queue command failed")
		return models.Queue{}, errors.New("Queue not found")
	}

	queue := parsedResult.Result.(models.Queue)

	// Para queues globales no hay nodo específico
	return queue, nil
}

func (bo *QueueBO) DeleteQueue(ctx context.Context, queueCode, vnamespace, cf, cfs string) error {
	writeCtx, writeCancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer writeCancel()

	deleteQueueCommand := &queue_command.DeleteQueueCommand{
		Code:       queueCode,
		VNamespace: vnamespace,
		CF:         cf,
		CFS:        cfs,
	}

	atstCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  deleteQueueCommand,
	}

	result, err := bo.Config.TenantNodesDictionary[cfs].Write(writeCtx, atstCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Str("QueueCode", queueCode).Str("VNamespace", vnamespace).Msg("Failed to delete queue")
		return errors.New("Failed to delete queue: " + err.Error())
	}

	buf := bytes.NewBuffer(result.Data)
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Str("QueueCode", queueCode).Str("VNamespace", vnamespace).Msg("Queue deletion command returned unexpected result type")
		return errors.New("Queue deletion command returned unexpected error")
	}

	if parsedResult.Error != "" {
		return errors.New("Failed to delete queue error: " + parsedResult.Error)
	}

	bo.Config.Logger.Info().Str("QueueCode", queueCode).Str("VNamespace", vnamespace).Msg("queue deleted successfully")
	return nil
}

func (bo *QueueBO) GetQueues(ctx context.Context, q string, cursor string, pageSize int, vNamespace string, includeHeaders bool, cf, cfs string) (db.FindResult[models.Queue], error) {
	paginateQueuesCommand := &queue_command.PaginateQueuesCommand{
		Query:          q,
		Cursor:         cursor,
		PageSize:       pageSize,
		VNamespace:     vNamespace,
		IncludeHeaders: includeHeaders,
		CF:             cf,
		CFS:            cfs,
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: paginateQueuesCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.TenantNodesDictionary[cfs].Read(readCtx, *queryCommand)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate queues command failed")
		return db.FindResult[models.Queue]{}, errors.New("Paginate queues failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate queues command failed")
		return db.FindResult[models.Queue]{}, errors.New("Paginate queues command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Paginate queues command failed")
		return db.FindResult[models.Queue]{}, errors.New("Paginate queues command failed")
	}

	findResult := parsedResult.Result.(db.FindResult[models.Queue])
	if findResult.Entities == nil {
		findResult.Entities = []models.Queue{}
	}

	return findResult, nil
}
