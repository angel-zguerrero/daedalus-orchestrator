package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/activity"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/usecase/command"
	activity_template_command "deadalus-orch/server/internal/usecase/command/activity-template"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	queue_command "deadalus-orch/server/internal/usecase/command/queue"
	scheduled_job_command "deadalus-orch/server/internal/usecase/command/scheduled-job"
	tenant_command "deadalus-orch/server/internal/usecase/command/tentant"
	workflow_execution_command "deadalus-orch/server/internal/usecase/command/workflow-execution"
	"deadalus-orch/shared/models"

	"github.com/rs/zerolog/log"
)

type ActivityJobMessage struct {
	ExecutionID  string                 `json:"executionId"`
	TokenID      string                 `json:"tokenId"`
	JobID        string                 `json:"jobId"`
	ActivityID   string                 `json:"activityId"`
	ActivityName string                 `json:"activityName,omitempty"`
	ActivityType string                 `json:"activityType,omitempty"`
	Input        map[string]interface{} `json:"input,omitempty"`
}

func (app *Application) StartWorkflowActivityWorker(interval time.Duration, batchSize int) {
	var workerLock sync.Mutex

	app.WorkflowActivityStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !workerLock.TryLock() {
					continue
				}

				go func() {
					defer workerLock.Unlock()

					if !app.MasterNodeIsReady || !app.MasterNodeIsLeader {
						return
					}

					select {
					case <-app.WorkflowActivityStopper.ShouldStop():
						return
					default:
					}

					app.processWorkflowActivityQueues(batchSize)
				}()

			case <-app.WorkflowActivityStopper.ShouldStop():
				log.Info().Msg("ℹ️ Workflow activity worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) processWorkflowActivityQueues(batchSize int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Process MasterNode global activity queues
	app.processActivityQueuesForNode(ctx, app.MasterNode, db.AdminFC, db.AdminFCSector, batchSize)

	// 2. Process TenantNodes activity queues
	cursor := ""
	pageSize := 50
	for {
		paginateTenantsCommand := &tenant_command.PaginateTenantsCommand{
			Cursor:   cursor,
			PageSize: pageSize,
		}

		queryCommand := &general_command.Query_Command{
			Command: &general_command.Repository_Command{
				CMD: paginateTenantsCommand,
			},
			Now: time.Now().UnixNano(),
		}

		result, err := app.MasterNode.Read(ctx, *queryCommand)
		if err != nil {
			break
		}

		tenantsResult, err := command.DecodeCommandResult[db.FindResult[models.TenantInMaster]](result.([]byte))
		if err != nil || len(tenantsResult.Entities) == 0 {
			break
		}

		for _, tenant := range tenantsResult.Entities {
			var tenantNode *dragonboat.RaftNode
			for i := range app.TenantNodes {
				if app.TenantNodes[i].ShardID == uint64(tenant.ShardId) {
					tenantNode = app.TenantNodes[i]
					break
				}
			}
			if tenantNode == nil {
				continue
			}

			cf := db.ColumnFamilyPrefix + fmt.Sprintf("%d", tenant.ColumnFamilyIndex)
			cfs := tenant.ID

			app.processActivityQueuesForNode(ctx, tenantNode, cf, cfs, batchSize)
		}

		if tenantsResult.Cursor == "" || len(tenantsResult.Entities) < pageSize {
			break
		}
		cursor = tenantsResult.Cursor
	}
}

func (app *Application) processActivityQueuesForNode(
	ctx context.Context,
	node *dragonboat.RaftNode,
	cf, cfs string,
	batchSize int,
) {
	if node == nil {
		return
	}

	cursor := ""
	for {
		pagCmd := &queue_command.PaginateQueuesCommand{
			PageSize:  batchSize,
			QueueType: models.WorkflowActivityQueue,
			Cursor:    cursor,
			CF:        cf,
			CFS:       cfs,
		}

		res, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.Queue]](
			node,
			ctx,
			pagCmd,
			config.GlobalConfiguration.ApiRaftTimeout,
			log.Logger,
			"paginate activity queues",
		)
		if err != nil || len(res.Entities) == 0 {
			break
		}

		autoComplete := config.GlobalConfiguration.AutoCompleteMockActivities

	for _, q := range res.Entities {
		if q.Type != models.WorkflowActivityQueue || q.State != models.QueueActive {
			continue
		}

		if q.MessagesCount == 0 {
			if q.CurrentDeliveringMessages == 0 && q.WorkflowDefinitionID != "" {
				reconcileCmd := &workflow_execution_command.ReconcilePendingWorkflowJobsCommand{
					WorkflowDefinitionID: q.WorkflowDefinitionID,
					QueueID:              q.ID,
					CF:                   cf,
					CFS:                  cfs,
				}
				recCtx, cancelRec := context.WithTimeout(context.Background(), 5*time.Second)
				recRes, recErr := dragonboat.ExecuteRepositoryCommand[workflow_execution_command.ReconcilePendingWorkflowJobsResult](
					node,
					recCtx,
					reconcileCmd,
					config.GlobalConfiguration.ApiRaftTimeout,
					log.Logger,
					"reconcile pending workflow activity jobs",
				)
				cancelRec()
				if recErr != nil || recRes.ReEnqueuedCount == 0 {
					continue
				}
				log.Info().
					Str("queueID", q.ID).
					Int("reEnqueuedCount", recRes.ReEnqueuedCount).
					Msg("🔄 Re-enqueued orphaned pending workflow activity jobs")
			} else {
				continue
			}
		}

		deqCmd := &queue_command.DequeueCommand{
			QueueID:       q.ID,
			JobWorkerID:   "WorkflowActivityWorker-1",
			LeaseDuration: 30 * time.Second,
			CF:            cf,
			CFS:           cfs,
		}

		deqRes, err := dragonboat.ExecuteRepositoryCommand[queue_command.DequeueResult](
			node,
			ctx,
			deqCmd,
			config.GlobalConfiguration.ApiRaftTimeout,
			log.Logger,
			"dequeue activity job message",
		)
		if err != nil || deqRes.Message.ID == "" {
			if err != nil {
				log.Debug().Err(err).Str("queueID", q.ID).Msg("⚠️ Failed to dequeue activity job message")
			}
			continue
		}

		log.Info().
			Str("queueID", q.ID).
			Str("messageID", deqRes.Message.ID).
			Msg("📥 Dequeued workflow activity job message")

		var jobMsg ActivityJobMessage
		if err := json.Unmarshal(deqRes.Message.Content, &jobMsg); err == nil && jobMsg.JobID != "" {
			var outputData map[string]interface{}
			var completedSuccessfully bool
			var lastExecErr error

			execCtx, cancelExec := context.WithTimeout(context.Background(), 30*time.Second)

			effectiveActivityType := jobMsg.ActivityType
			effectiveInput := jobMsg.Input

			startTemplateCode := strings.TrimSpace(jobMsg.ActivityType)
			if jobMsg.Input != nil {
				if tc, ok := jobMsg.Input["_templateCode"].(string); ok && strings.TrimSpace(tc) != "" {
					startTemplateCode = strings.TrimSpace(tc)
				}
			}

			tplChain, resolvedBaseType := app.resolveActivityTemplateChain(execCtx, startTemplateCode, jobMsg.ActivityType, node, cf, cfs)
			if len(tplChain) > 0 {
				effectiveInput = mergeTemplateChainInput(tplChain, effectiveInput)
			}
			if resolvedBaseType != "" {
				effectiveActivityType = resolvedBaseType
			}

			if db.NormalizeBuiltinActivityType(effectiveActivityType) == "" {
				if inferred := db.InferBaseActivityTypeFromMap(effectiveInput); inferred != "" {
					effectiveActivityType = inferred
				} else if db.NormalizeBuiltinActivityType(jobMsg.ActivityType) != "" {
					effectiveActivityType = jobMsg.ActivityType
				} else {
					effectiveActivityType = "task"
				}
			}

			nativeExec, foundNative := activity.GetRegistry().Get(effectiveActivityType)

			if foundNative {
				out, err := nativeExec.Execute(execCtx, effectiveInput)
				cancelExec()

				if err == nil {
					outputData = out
					completedSuccessfully = true
					log.Info().
						Str("jobID", jobMsg.JobID).
						Str("activityType", jobMsg.ActivityType).
						Str("effectiveActivityType", effectiveActivityType).
						Msg("🚀 Successfully executed activity natively in Daedalus WORKER_ACTIVITIES")
				} else {
					lastExecErr = err
					log.Warn().
						Err(err).
						Str("jobID", jobMsg.JobID).
						Str("activityType", jobMsg.ActivityType).
						Msg("⚠️ Native activity execution failed")
				}
			} else {
				cancelExec()
				if autoComplete {
					outputData = map[string]interface{}{"status": "SUCCESS"}
					completedSuccessfully = true
					log.Info().
						Str("jobID", jobMsg.JobID).
						Str("activityType", jobMsg.ActivityType).
						Msg("⚡ Auto-completed activity job in Daedalus WORKER_ACTIVITIES")
				} else {
					lastExecErr = fmt.Errorf("no native executor found for activity type %q", effectiveActivityType)
					log.Warn().
						Str("jobID", jobMsg.JobID).
						Str("activityType", jobMsg.ActivityType).
						Msg("⚠️ No native executor found for activity type")
				}
			}

			if !completedSuccessfully && lastExecErr != nil {
				failCmd := &workflow_execution_command.HandleJobFailureCommand{
					JobID:    jobMsg.JobID,
					WorkerID: "WorkflowActivityWorker-1",
					ErrorMsg: lastExecErr.Error(),
					CF:       cf,
					CFS:      cfs,
				}
				failCtx, cancelFail := context.WithTimeout(context.Background(), 10*time.Second)
				failRes, failErr := dragonboat.ExecuteRepositoryCommand[workflow_execution_command.HandleJobFailureResult](
					node,
					failCtx,
					failCmd,
					config.GlobalConfiguration.ApiRaftTimeout,
					log.Logger,
					"handle activity job failure",
				)
				cancelFail()

				if failErr == nil && failRes.RetriesExceeded {
					log.Error().
						Str("jobID", jobMsg.JobID).
						Int32("retries", failRes.CurrentRetries).
						Int32("maxRetries", failRes.MaxRetries).
						Msg("❌ Activity job exceeded max retries. Marked job and workflow execution as FAILED.")

					ackCmd := &queue_command.AckMessageCommand{
						LeaseID: deqRes.Lease.ID,
						CF:      cf,
						CFS:     cfs,
					}
					ackCtx, cancelAck := context.WithTimeout(context.Background(), 10*time.Second)
					_, _ = dragonboat.ExecuteRepositoryCommand[queue_command.AckMessageResult](
						node,
						ackCtx,
						ackCmd,
						config.GlobalConfiguration.ApiRaftTimeout,
						log.Logger,
						"ack failed activity message",
					)
					cancelAck()
				} else if failErr == nil {
					log.Warn().
						Str("jobID", jobMsg.JobID).
						Int32("retries", failRes.CurrentRetries).
						Int32("maxRetries", failRes.MaxRetries).
						Msg("⏳ Activity job retry count incremented. Scheduling retry message with 5s delay via ScheduledJob.")

					nextRunAt := time.Now().Add(5 * time.Second)
					retryID := fmt.Sprintf("retry_%s_%d_%d", jobMsg.JobID, failRes.CurrentRetries, time.Now().UnixNano())
					schedCmd := &scheduled_job_command.CreateScheduledJobCommand{
						ScheduledJob: models.ScheduledJob{
							ID:          retryID,
							Code:        retryID,
							TenantID:    cfs,
							TargetType:  "queue",
							TargetID:    q.ID,
							VNamespace:  deqRes.Message.VNamespace,
							Content:     string(deqRes.Message.Content),
							ContentType: "application/json",
							Type:        models.ScheduledJobOneOff,
							RunAt:       &nextRunAt,
							NextRunAt:   nextRunAt,
						},
						CF:  cf,
						CFS: cfs,
					}
					schedCtx, cancelSched := context.WithTimeout(context.Background(), 10*time.Second)
					_, schedErr := dragonboat.ExecuteRepositoryCommand[interface{}](
						node,
						schedCtx,
						schedCmd,
						config.GlobalConfiguration.ApiRaftTimeout,
						log.Logger,
						"schedule activity retry job message",
					)
					cancelSched()

					if schedErr != nil {
						// Fallback: immediately re-enqueue into the activity queue so the job is never orphaned in PENDING
						reEnqMsgID := fmt.Sprintf("reenq_%s_%d_%d", jobMsg.JobID, failRes.CurrentRetries, time.Now().UnixNano())
						reEnqCmd := &queue_command.EnqueueCommand{
							Messages: []models.QueueMessage{
								{
									ID:          reEnqMsgID,
									MessageID:   reEnqMsgID,
									QueueID:     q.ID,
									VNamespace:  deqRes.Message.VNamespace,
									Content:     deqRes.Message.Content,
									ContentType: "application/json",
									CreatedAt:   time.Now(),
									UpdatedAt:   time.Now(),
								},
							},
							CF:  cf,
							CFS: cfs,
						}
						enqCtx, cancelEnq := context.WithTimeout(context.Background(), 10*time.Second)
						_, _ = dragonboat.ExecuteRepositoryCommand[interface{}](
							node,
							enqCtx,
							reEnqCmd,
							config.GlobalConfiguration.ApiRaftTimeout,
							log.Logger,
							"fallback re-enqueue activity retry message",
						)
						cancelEnq()
					}

					ackCmd := &queue_command.AckMessageCommand{
						LeaseID: deqRes.Lease.ID,
						CF:      cf,
						CFS:     cfs,
					}
					ackCtx, cancelAck := context.WithTimeout(context.Background(), 10*time.Second)
					_, _ = dragonboat.ExecuteRepositoryCommand[queue_command.AckMessageResult](
						node,
						ackCtx,
						ackCmd,
						config.GlobalConfiguration.ApiRaftTimeout,
						log.Logger,
						"ack activity message for retry",
					)
					cancelAck()
				} else {
					log.Error().Err(failErr).Str("jobID", jobMsg.JobID).Msg("❌ Failed to execute HandleJobFailureCommand")
				}
			}

			if completedSuccessfully {
				compCmd := &workflow_execution_command.CompleteJobCommand{
					JobID:      jobMsg.JobID,
					WorkerID:   "WorkflowActivityWorker-1",
					OutputData: outputData,
					CF:         cf,
					CFS:        cfs,
				}
				compCtx, cancelComp := context.WithTimeout(context.Background(), 10*time.Second)
				_, compErr := dragonboat.ExecuteRepositoryCommand[interface{}](
					node,
					compCtx,
					compCmd,
					config.GlobalConfiguration.ApiRaftTimeout,
					log.Logger,
					"complete job",
				)
				cancelComp()
				if compErr != nil {
					log.Error().Err(compErr).
						Str("jobID", jobMsg.JobID).
						Msg("❌ Failed to complete activity job")
				} else {
					log.Info().
						Str("jobID", jobMsg.JobID).
						Msg("✅ Successfully completed workflow activity job")
				}

				ackCmd := &queue_command.AckMessageCommand{
					LeaseID: deqRes.Lease.ID,
					CF:      cf,
					CFS:     cfs,
				}
				ackCtx, cancelAck := context.WithTimeout(context.Background(), 10*time.Second)
				_, ackErr := dragonboat.ExecuteRepositoryCommand[queue_command.AckMessageResult](
					node,
					ackCtx,
					ackCmd,
					config.GlobalConfiguration.ApiRaftTimeout,
					log.Logger,
					"ack activity message",
				)
				cancelAck()
				if ackErr != nil {
					log.Warn().Err(ackErr).Str("leaseID", deqRes.Lease.ID).Msg("⚠️ Failed to ack activity message")
				}
			}
		}
	}

	if res.Cursor == "" || len(res.Entities) < batchSize {
		break
	}
	cursor = res.Cursor
	}
}

func (app *Application) resolveActivityTemplate(
	ctx context.Context,
	activityType string,
	node *dragonboat.RaftNode,
	cf, cfs string,
) *models.ActivityTemplate {
	trimmed := strings.TrimSpace(activityType)
	if trimmed == "" || db.IsBuiltinActivityType(trimmed) {
		return nil
	}

	// 1. Try current node (tenant node if tenant queue, or MasterNode if global queue)
	if node != nil {
		cmd := &activity_template_command.GetActivityTemplateByCodeOrIDCommand{
			CodeOrID: trimmed,
			CF:       cf,
			CFS:      cfs,
		}
		res, err := dragonboat.ExecuteRepositoryQuery[models.ActivityTemplate](
			node,
			ctx,
			cmd,
			config.GlobalConfiguration.ApiRaftTimeout,
			log.Logger,
			"resolve activity template in current node",
		)
		if err == nil && res.ID != "" {
			return &res
		}
	}

	// 2. Fallback to MasterNode (Global Activity Templates) if current node was a tenant node
	if app.MasterNode != nil && (node != app.MasterNode || cf != db.AdminFC) {
		cmd := &activity_template_command.GetActivityTemplateByCodeOrIDCommand{
			CodeOrID: trimmed,
			CF:       db.AdminFC,
			CFS:      db.AdminFCSector,
		}
		res, err := dragonboat.ExecuteRepositoryQuery[models.ActivityTemplate](
			app.MasterNode,
			ctx,
			cmd,
			config.GlobalConfiguration.ApiRaftTimeout,
			log.Logger,
			"resolve global activity template",
		)
		if err == nil && res.ID != "" {
			return &res
		}
	}

	return nil
}

// resolveActivityTemplateChain walks up ParentTemplateId up to 10 levels, returning the ancestor chain ordered
// from root parent to leaf child ([Activity 1, Activity 2, ...]) and the resolved built-in base activity type.
func (app *Application) resolveActivityTemplateChain(
	ctx context.Context,
	startTemplateCode string,
	fallbackActivityType string,
	node *dragonboat.RaftNode,
	cf, cfs string,
) ([]*models.ActivityTemplate, string) {
	var chainRootToLeaf []*models.ActivityTemplate
	visited := make(map[string]bool)
	currCode := strings.TrimSpace(startTemplateCode)
	effectiveBaseType := strings.TrimSpace(fallbackActivityType)

	for depth := 0; depth < 10 && currCode != ""; depth++ {
		lowerCode := strings.ToLower(currCode)
		if visited[lowerCode] {
			break
		}
		visited[lowerCode] = true

		if canonical := db.NormalizeBuiltinActivityType(currCode); canonical != "" {
			effectiveBaseType = canonical
			break
		}

		tpl := app.resolveActivityTemplate(ctx, currCode, node, cf, cfs)
		if tpl == nil {
			break
		}

		// Prepend so chainRootToLeaf is ordered [RootAncestor, ..., LeafChild]
		chainRootToLeaf = append([]*models.ActivityTemplate{tpl}, chainRootToLeaf...)

		// Directly inspect RootActivity on the template
		if canonicalRoot := db.NormalizeBuiltinActivityType(tpl.RootActivity); canonicalRoot != "" {
			effectiveBaseType = canonicalRoot
		}

		nextParent := strings.TrimSpace(tpl.ParentTemplateId)
		if nextParent != "" {
			currCode = nextParent
		} else {
			if db.NormalizeBuiltinActivityType(effectiveBaseType) == "" {
				if inferred := db.InferBaseActivityTypeFromPayload(tpl.Payload); inferred != "" {
					effectiveBaseType = inferred
				}
			}
			break
		}
	}

	if canonical := db.NormalizeBuiltinActivityType(effectiveBaseType); canonical != "" {
		effectiveBaseType = canonical
	} else if len(chainRootToLeaf) > 0 {
		for _, tpl := range chainRootToLeaf {
			if canonicalRoot := db.NormalizeBuiltinActivityType(tpl.RootActivity); canonicalRoot != "" {
				effectiveBaseType = canonicalRoot
				break
			}
		}
	}

	return chainRootToLeaf, effectiveBaseType
}

func mergeTemplateInput(template *models.ActivityTemplate, jobInput map[string]interface{}) map[string]interface{} {
	if template == nil {
		return db.MergeActivityTemplateHierarchy(nil, jobInput)
	}
	return db.MergeActivityTemplateHierarchy([]*models.ActivityTemplate{template}, jobInput)
}

func mergeTemplateChainInput(chainRootToLeaf []*models.ActivityTemplate, jobInput map[string]interface{}) map[string]interface{} {
	return db.MergeActivityTemplateHierarchy(chainRootToLeaf, jobInput)
}


