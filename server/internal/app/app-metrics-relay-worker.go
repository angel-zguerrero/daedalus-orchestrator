package app

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	metrics_command "deadalus-orch/server/internal/usecase/command/metrics"
	tentant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartMetricsRelayWorker(interval time.Duration) {
	app.MetricsRelayWorkerStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !app.MasterNodeIsReady {
					log.Debug().Msg("⏳ MetricsRelay worker is waiting for the master node to be ready")
					continue
				}

				if !app.MasterNodeIsLeader {
					log.Debug().Msg("⏳ MetricsRelay worker is waiting for the master node to be leader")
					continue
				}

				select {
				case <-app.MetricsRelayWorkerStopper.ShouldStop():
					log.Info().Msg("🛑 MetricsRelay worker received stop signal before execution")
					return
				default:
				}

				go func() {
					app.processMetricsRelays()
				}()

			case <-app.MetricsRelayWorkerStopper.ShouldStop():
				log.Info().Msg("ℹ️  MetricsRelay worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) processMetricsRelays() {
	now := time.Now()

	for _, tenantNode := range app.TenantNodes {
		select {
		case <-app.MetricsRelayWorkerStopper.ShouldStop():
			log.Info().Msg("🛑 MetricsRelay worker received stop signal during processing")
			return
		default:
		}

		go func(node *dragonboat.RaftNode) {
			app.processMetricsEventsForNode(node, now)
		}(tenantNode)
	}
}

func (app *Application) processMetricsEventsForNode(tenantNode *dragonboat.RaftNode, now time.Time) {
	// Query outbox events for this node
	getEventsCmd := &tentant_command.GetOutboxEventsCommand{
		CFS: "", // Not used since OutboxEventRepository is global
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: getEventsCmd,
		},
		Now: now.UnixNano(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := tenantNode.Read(ctx, *queryCommand)
	if err != nil {
		log.Err(err).Uint64("shard_id", tenantNode.ShardID).Msg("❌ Failed to read outbox events for metrics relay")
		return
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		log.Err(err).Uint64("shard_id", tenantNode.ShardID).Msg("❌ Failed to decode outbox events for metrics relay")
		return
	}

	if parsedResult.Error != "" {
		log.Error().Str("error", parsedResult.Error).Uint64("shard_id", tenantNode.ShardID).Msg("❌ Command error reading outbox events")
		return
	}

	events, ok := parsedResult.Result.([]models.OutboxEvent)
	if !ok || len(events) == 0 {
		return // No events
	}

	var processedEventIDs []string
	var allRelayedBuckets []models.MetricsBucket

	for _, event := range events {
		if event.EventType == models.EventTypeMetricsRelay {
			var buckets []models.MetricsBucket
			if err := json.Unmarshal(event.Payload, &buckets); err != nil {
				log.Err(err).Str("event_id", event.ID).Msg("❌ Failed to unmarshal metrics payload from OutboxEvent")
				// We still append the event ID to delete it so it doesn't block the queue forever
			} else {
				allRelayedBuckets = append(allRelayedBuckets, buckets...)
			}
			processedEventIDs = append(processedEventIDs, event.ID)
		}
	}

	if len(processedEventIDs) == 0 {
		return
	}

	log.Debug().Uint64("shard_id", tenantNode.ShardID).Int("events_count", len(processedEventIDs)).Int("buckets_count", len(allRelayedBuckets)).Msg("📬 Relaying metric buckets to Master Node")

	if len(allRelayedBuckets) > 0 {
		saveMetricsCmd := &metrics_command.SaveMetricsBucketsCommand{
			Buckets: allRelayedBuckets,
			IsRelay: true,
		}

		fsmCmd := general_command.FSM_Command{
			Now:  utils.GetNowInInt(),
			Type: general_command.REPOSITORY_COMMAND,
			CMD:  saveMetricsCmd,
		}

		ctxWrite, cancelWrite := context.WithTimeout(context.Background(), 10*time.Second)
		resultChan, err := app.MasterNode.Write(ctxWrite, fsmCmd)
		if err != nil {
			cancelWrite()
			log.Err(err).Msg("❌ Failed to submit SaveMetricsBuckets to Master")
			return // Stop processing so we don't delete the events if we failed to relay!
		}
		
		select {
		case writeResult := <-resultChan:
			cancelWrite()
			if writeResult.Error != nil {
				log.Err(writeResult.Error).Msg("❌ Failed to execute SaveMetricsBuckets in Master")
				return
			}
		case <-ctxWrite.Done():
			cancelWrite()
			log.Err(ctxWrite.Err()).Msg("❌ Timeout saving metric buckets in Master")
			return
		}
	}

	// Delete processed events from Shard Node
	deleteCmd := &tentant_command.DeleteOutboxEventsCommand{
		EventIDs: processedEventIDs,
		CFS:      "", // Not used
	}

	deleteFsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  deleteCmd,
	}

	ctxDel, cancelDel := context.WithTimeout(context.Background(), 10*time.Second)
	delResultChan, err := tenantNode.Write(ctxDel, deleteFsmCmd)
	if err != nil {
		cancelDel()
		log.Err(err).Uint64("shard_id", tenantNode.ShardID).Msg("❌ Failed to submit delete outbox events for metrics relay")
		return
	}
	
	select {
	case writeResult := <-delResultChan:
		cancelDel()
		if writeResult.Error != nil {
			log.Err(writeResult.Error).Uint64("shard_id", tenantNode.ShardID).Msg("❌ Failed to delete outbox events for metrics relay")
		} else {
			log.Debug().Uint64("shard_id", tenantNode.ShardID).Int("deleted", len(processedEventIDs)).Msg("✅ Metrics outbox events relayed and deleted")
		}
	case <-ctxDel.Done():
		cancelDel()
		log.Err(ctxDel.Err()).Uint64("shard_id", tenantNode.ShardID).Msg("❌ Timeout deleting metrics outbox events")
	}
}
