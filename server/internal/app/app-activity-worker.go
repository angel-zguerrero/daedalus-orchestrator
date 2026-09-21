package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/activity"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/usecase/command"
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

func invokeExternalCamundaRunner(ctx context.Context, runnerURL string, jobMsg ActivityJobMessage) (map[string]interface{}, error) {
	targetURL := runnerURL
	parsed, err := url.Parse(runnerURL)
	if err == nil && (parsed.Path == "" || parsed.Path == "/") {
		targetURL = strings.TrimRight(runnerURL, "/") + "/execute"
	}

	payloadBytes, err := json.Marshal(jobMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("connector runner (%s) returned status code %d: %s", targetURL, resp.StatusCode, string(bodyBytes))
	}

	var output map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		output = map[string]interface{}{
			"status": "SUCCESS",
		}
	}
	return output, nil
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

	pagCmd := &queue_command.PaginateQueuesCommand{
		PageSize:  batchSize,
		QueueType: models.WorkflowActivityQueue,
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
		return
	}

	runnerURL := config.GlobalConfiguration.CamundaConnectorRunnerURL
	autoComplete := config.GlobalConfiguration.AutoCompleteMockActivities

	for _, q := range res.Entities {
		if q.Type != models.WorkflowActivityQueue || q.State != models.QueueActive || q.MessagesCount == 0 {
			continue
		}

		if runnerURL == "" && !autoComplete {
			log.Debug().
				Str("queueID", q.ID).
				Int("pendingMessages", q.MessagesCount).
				Msg("⏳ Activity job(s) pending in queue for external Camunda connector runner (configure camunda_connector_runner_url or auto_complete_mock_activities=true)")
			continue
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
			nativeExec, foundNative := activity.GetRegistry().Get(jobMsg.ActivityType)

			if foundNative {
				out, err := nativeExec.Execute(execCtx, jobMsg.Input)
				cancelExec()

				if err == nil {
					outputData = out
					completedSuccessfully = true
					log.Info().
						Str("jobID", jobMsg.JobID).
						Str("activityType", jobMsg.ActivityType).
						Msg("🚀 Successfully executed activity natively in Daedalus WORKER_ACTIVITIES")
				} else {
					lastExecErr = err
					log.Warn().
						Err(err).
						Str("jobID", jobMsg.JobID).
						Str("activityType", jobMsg.ActivityType).
						Msg("⚠️ Native activity execution failed")
				}
			} else if runnerURL != "" {
				out, err := invokeExternalCamundaRunner(execCtx, runnerURL, jobMsg)
				cancelExec()

				if err == nil {
					outputData = out
					completedSuccessfully = true
					log.Info().
						Str("jobID", jobMsg.JobID).
						Str("activityType", jobMsg.ActivityType).
						Msg("🚀 Successfully executed job via external activity runner")
				} else {
					lastExecErr = err
					log.Warn().
						Err(err).
						Str("jobID", jobMsg.JobID).
						Str("activityType", jobMsg.ActivityType).
						Msg("⚠️ External activity runner execution failed")
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

					nextRunAt := time.Now().Add(5 * time.Second)
					schedCmd := &scheduled_job_command.CreateScheduledJobCommand{
						ScheduledJob: models.ScheduledJob{
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
					_, _ = dragonboat.ExecuteRepositoryCommand[interface{}](
						node,
						schedCtx,
						schedCmd,
						config.GlobalConfiguration.ApiRaftTimeout,
						log.Logger,
						"schedule activity retry job message",
					)
					cancelSched()
				} else {
					log.Error().Err(failErr).Str("jobID", jobMsg.JobID).Msg("❌ Failed to execute HandleJobFailureCommand")
				}
			}

			if !completedSuccessfully && autoComplete {
				outputData = map[string]interface{}{
					"executedBy": "InternalActivityWorker",
					"status":     "SUCCESS",
				}
				completedSuccessfully = true
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
		} else {
			log.Error().
				Str("content", string(deqRes.Message.Content)).
				Msg("❌ Failed to unmarshal ActivityJobMessage or missing fields")
		}
	}
}
