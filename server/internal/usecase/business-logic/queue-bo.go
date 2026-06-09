package business_logic

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
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

// ValidateQueueFields validates queue fields and their relationships
func (bo *QueueBO) ValidateQueueFields(queue *models.Queue) error {
	// Validate DefaultQueueMessageTTL >= 0
	if queue.DefaultQueueMessageTTL < 0 {
		return errors.New("DefaultQueueMessageTTL cannot be negative")
	}

	// Validate DefaultQueueMessageDelayTime >= 0
	if queue.DefaultQueueMessageDelayTime < 0 {
		return errors.New("DefaultQueueMessageDelayTime cannot be negative")
	}

	// Validate QueueExpires >= 0
	if queue.QueueExpires < 0 {
		return errors.New("QueueExpires cannot be negative")
	}

	// ExpireAt should only be set if DefaultQueueMessageDelayTime > 0
	if queue.QueueExpires > 0 {
		// Calculate ExpireAt based on current time + DefaultQueueMessageDelayTime
		expireTime := time.Now().Add(time.Duration(queue.QueueExpires) * time.Second)
		fmt.Println("Calculated ExpireAt:", expireTime)
		fmt.Println("Current time:", time.Now())
		queue.ExpireAt = &expireTime
	}

	return nil
}

func (bo *QueueBO) CreateQueue(ctx context.Context, code, vnamespace, name string, queueType models.QueueType, headers map[string]string, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) (models.Queue, error) {
	queue := &models.Queue{
		ID:                           strings.ReplaceAll(uuid.New().String(), "-", ""),
		Code:                         code,
		Name:                         name,
		Type:                         queueType,
		VNamespace:                   vnamespace,
		State:                        models.QueueActive, // Default state
		DefaultQueueMessageTTL:       0,                  // Default TTL
		DefaultQueueMessageDelayTime: 0,                  // Default delay time
		QueueExpires:                 0,                  // Default queue expires
		ExpireAt:                     nil,                // Default nil
		AllowDuplicated:              true,               // Default allow duplicated
		MaxAttempts:                  1,                  // Default max attempts
		Headers:                      headers,            // Add headers support
	}

	createdList, err := bo.BulkCreateQueue(ctx, []*models.Queue{queue}, cf, cfs, tenant, tenantNode)
	if err != nil {
		return models.Queue{}, err
	}
	return createdList[0], nil
}

func (bo *QueueBO) BulkCreateQueue(ctx context.Context, queues []*models.Queue, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) ([]models.Queue, error) {
	if tenant.Status == models.PendingForDeletion {
		return nil, errors.New("cannot create queue when tenant is pending for deletion")
	}

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

		// Validate all required fields
		if err := bo.ValidateQueueFields(t); err != nil {
			return nil, fmt.Errorf("validation failed for queue %s: %w", t.Code, err)
		}
	}

	assertQueueCommand := &queue_command.AssertQueueCommand{
		Queues: make([]models.Queue, len(queues)),
		CF:     cf,
		CFS:    cfs,
	}
	for i, t := range queues {
		assertQueueCommand.Queues[i] = *t
	}

	created, err := dragonboat.ExecuteRepositoryCommand[[]models.Queue](
		tenantNode,
		ctx,
		assertQueueCommand,
		config.GlobalConfiguration.ApiRaftTimeout*time.Duration(len(queues)),
		bo.Config.Logger,
		"bulk create queues",
	)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (bo *QueueBO) GetQueue(ctx context.Context, queueCode, vnamespace string, includeHeaders bool, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) (models.Queue, error) {
	findQueueCommand := &queue_command.FindQueueCommand{
		Code:           queueCode,
		VNamespace:     vnamespace,
		IncludeHeaders: includeHeaders,
		CF:             cf,
		CFS:            cfs,
	}

	queue, err := dragonboat.ExecuteRepositoryQuery[models.Queue](
		tenantNode,
		ctx,
		findQueueCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"find queue",
	)
	if err != nil {
		if strings.Contains(err.Error(), "entity not found") {
			return models.Queue{}, errors.New("Queue not found")
		}
		return models.Queue{}, fmt.Errorf("find queue command failed: %w", err)
	}

	return queue, nil
}

func (bo *QueueBO) DeleteQueue(ctx context.Context, queueCode, vnamespace, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) error {
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

	resultChan, err := tenantNode.Write(writeCtx, atstCmd)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Str("QueueCode", queueCode).Str("VNamespace", vnamespace).Msg("Failed to start delete queue operation")
		return errors.New("Failed to start delete queue operation: " + err.Error())
	}

	// Wait for the result since we need to process it
	var writeResult dragonboat.WriteResult
	select {
	case writeResult = <-resultChan:
		if writeResult.Error != nil {
			bo.Config.Logger.Error().Err(writeResult.Error).Str("QueueCode", queueCode).Str("VNamespace", vnamespace).Msg("Failed to delete queue")
			return errors.New("Failed to delete queue: " + writeResult.Error.Error())
		}
	case <-writeCtx.Done():
		bo.Config.Logger.Error().Err(writeCtx.Err()).Str("QueueCode", queueCode).Str("VNamespace", vnamespace).Msg("Delete queue operation timed out")
		return errors.New("Delete queue operation timed out: " + writeCtx.Err().Error())
	}

	buf := bytes.NewBuffer(writeResult.Result.Data)
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

func (bo *QueueBO) GetQueues(ctx context.Context, q string, cursor string, pageSize int, vNamespace string, includeHeaders bool, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode, includeSupervisorInfo bool) (db.FindResult[models.Queue], error) {
	paginateQueuesCommand := &queue_command.PaginateQueuesCommand{
		Query:          q,
		Cursor:         cursor,
		PageSize:       pageSize,
		VNamespace:     vNamespace,
		IncludeHeaders: includeHeaders,
		CF:             cf,
		CFS:            cfs,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.Queue]](
		tenantNode,
		ctx,
		paginateQueuesCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"paginate queues",
	)
	if err != nil {
		return db.FindResult[models.Queue]{}, fmt.Errorf("paginate queues failed: %w", err)
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.Queue{}
	}



	return findResult, nil
}

// GetQueuesWithFilter paginates queues using DB-level filter rules derived from a ClaimWorkFilter.
// Only queues with MessagesCount > 0 are returned. The vNamespace filter and exact code
// exclusions are pushed to the repository; ExcludeQueuePatterns are applied in memory inside
// the repository.
func (bo *QueueBO) GetQueuesWithFilter(ctx context.Context, filter models.ClaimWorkFilter, cursor string, pageSize int, vNamespace string, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) (db.FindResult[models.Queue], error) {
	cmd := &queue_command.PaginateQueuesWithFilterCommand{
		Filter:     filter,
		VNamespace: vNamespace,
		Cursor:     cursor,
		PageSize:   pageSize,
		CF:         cf,
		CFS:        cfs,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.Queue]](
		tenantNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"paginate queues with filter",
	)
	if err != nil {
		return db.FindResult[models.Queue]{}, fmt.Errorf("paginate queues with filter failed: %w", err)
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.Queue{}
	}

	return findResult, nil
}

func (bo *QueueBO) GetQueuesBySupervisionState(ctx context.Context, q string, cursor string, pageSize int, vNamespace string, includeHeaders bool, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode, supervisionState models.QueueSupervisionState, includeSupervisorInfo bool) (db.FindResult[models.Queue], error) {
	paginateQueuesCommand := &queue_command.PaginateQueuesCommand{
		Query:            q,
		Cursor:           cursor,
		PageSize:         pageSize,
		VNamespace:       vNamespace,
		IncludeHeaders:   includeHeaders,
		CF:               cf,
		CFS:              cfs,
		SupervisionState: supervisionState,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.Queue]](
		tenantNode,
		ctx,
		paginateQueuesCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"paginate queues",
	)
	if err != nil {
		return db.FindResult[models.Queue]{}, fmt.Errorf("paginate queues failed: %w", err)
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.Queue{}
	}



	return findResult, nil
}

func (bo *QueueBO) EnqueueMessage(ctx context.Context, queueCode string, message models.QueueMessage, vnamespace string, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) (string, error) {
	// First, get the queue to ensure it exists
	queue, err := bo.GetQueue(ctx, queueCode, vnamespace, false, cf, cfs, tenant, tenantNode)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Failed to get queue")
		return "", fmt.Errorf("failed to get queue: %w", err)
	}

	// Check if queue is in active state
	if queue.State != models.QueueActive {
		bo.Config.Logger.Info().Str("queueCode", queueCode).Str("queueState", string(queue.State)).Msg("Queue is not in active state - skipping message enqueue")
		return "", fmt.Errorf("queue '%s' is not in active state (current state: %s)", queueCode, queue.State)
	}

	// Generate message ID if not provided
	if message.MessageID == "" {
		message.MessageID = strings.ReplaceAll(uuid.New().String(), "-", "")
	}

	// Create the queue message with the queue ID
	queueMessage := models.QueueMessage{
		ID:          strings.ReplaceAll(uuid.New().String(), "-", ""),
		MessageID:   message.MessageID,
		Content:     message.Content,
		ContentType: message.ContentType,
		Headers:     message.Headers,
		QueueID:     queue.ID,
		Priority:    message.Priority,
		Handler:     message.Handler,
		Parameters:  message.Parameters,
		VNamespace:  vnamespace,
	}

	// Enqueue the message
	enqueueCommand := &queue_command.EnqueueCommand{
		Messages: []models.QueueMessage{queueMessage},
		CF:       cf,
		CFS:      cfs,
	}

	createdMessages, err := dragonboat.ExecuteRepositoryCommand[[]models.QueueMessage](
		tenantNode,
		ctx,
		enqueueCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"enqueue message",
	)
	if err != nil {
		return "", err
	}

	if len(createdMessages) > 0 {
		return createdMessages[0].ID, nil
	}

	return "", fmt.Errorf("no message was created")
}


// DequeueMessage removes the next available message from the queue according to the
// threshold-based fair priority queue algorithm, creates a QueueMessageLease bound to
// jobWorkerID, and returns both the message and the lease.
//
// The lease expiry (LeaseUntil) is calculated as:
//
//	now + config.GlobalConfiguration.MessageLeaseDuration
//
// where MessageLeaseDuration follows the same precedence as all other configuration
// values: command-line flag > environment variable (MESSAGE_LEASE_DURATION) >
// configuration file (message_lease_duration key, in seconds) > built-in default (30 s).
func (bo *QueueBO) GetQueueMessages(ctx context.Context, queueCode, vnamespace string, cursor string, pageSize int, cf, cfs string, tenant *models.TenantInMaster, tenantNode *dragonboat.RaftNode) (db.FindResult[queue_command.QueueMessageWithLease], error) {
	queue, err := bo.GetQueue(ctx, queueCode, vnamespace, false, cf, cfs, tenant, tenantNode)
	if err != nil {
		return db.FindResult[queue_command.QueueMessageWithLease]{}, fmt.Errorf("failed to get queue: %w", err)
	}

	paginateCmd := &queue_command.PaginateQueueMessagesCommand{
		QueueID:  queue.ID,
		Cursor:   cursor,
		PageSize: pageSize,
		CF:       cf,
		CFS:      cfs,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[queue_command.QueueMessageWithLease]](
		tenantNode,
		ctx,
		paginateCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"paginate queue messages",
	)
	if err != nil {
		return db.FindResult[queue_command.QueueMessageWithLease]{}, fmt.Errorf("paginate queue messages failed: %w", err)
	}

	if findResult.Entities == nil {
		findResult.Entities = []queue_command.QueueMessageWithLease{}
	}

	return findResult, nil
}

func (bo *QueueBO) DequeueMessage(
	ctx context.Context,
	queueCode string,
	vnamespace string,
	jobWorkerID string,
	cf, cfs string,
	tenant *models.TenantInMaster,
	tenantNode *dragonboat.RaftNode,
) (queue_command.DequeueResult, error) {
	if jobWorkerID == "" {
		return queue_command.DequeueResult{}, errors.New("jobWorkerID is required")
	}

	// Resolve the queue to obtain its internal ID.
	queue, err := bo.GetQueue(ctx, queueCode, vnamespace, false, cf, cfs, tenant, tenantNode)
	if err != nil {
		return queue_command.DequeueResult{}, fmt.Errorf("failed to get queue: %w", err)
	}

	dequeueCmd := &queue_command.DequeueCommand{
		QueueID:       queue.ID,
		JobWorkerID:   jobWorkerID,
		LeaseDuration: config.GlobalConfiguration.MessageLeaseDuration,
		CF:            cf,
		CFS:           cfs,
	}

	result, err := dragonboat.ExecuteRepositoryCommand[queue_command.DequeueResult](
		tenantNode,
		ctx,
		dequeueCmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"dequeue message",
	)
	if err != nil {
		return queue_command.DequeueResult{}, fmt.Errorf("dequeue command failed: %w", err)
	}

	return result, nil
}
