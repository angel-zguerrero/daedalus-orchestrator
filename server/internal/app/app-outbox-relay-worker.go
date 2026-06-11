package app

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	tentant_command "deadalus-orch/server/internal/usecase/command/tentant"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartOutboxRelayWorker(interval time.Duration) {
	app.OutboxRelayWorkerStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !app.MasterNodeIsReady {
					log.Debug().Msg("⏳ OutboxRelay worker is waiting for the master node to be ready")
					continue
				}

				if !app.MasterNodeIsLeader {
					log.Debug().Msg("⏳ OutboxRelay worker is waiting for the master node to be leader")
					continue
				}

				select {
				case <-app.OutboxRelayWorkerStopper.ShouldStop():
					log.Info().Msg("🛑 OutboxRelay worker received stop signal before execution")
					return
				default:
				}

				go func() {
					app.processOutboxRelays()
				}()

			case <-app.OutboxRelayWorkerStopper.ShouldStop():
				log.Info().Msg("ℹ️  OutboxRelay worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) processOutboxRelays() {
	now := time.Now()

	for _, tenantNode := range app.TenantNodes {
		select {
		case <-app.OutboxRelayWorkerStopper.ShouldStop():
			log.Info().Msg("🛑 OutboxRelay worker received stop signal during processing")
			return
		default:
		}

		go func(node *dragonboat.RaftNode) {
			app.processOutboxEventsForNode(node, now)
		}(tenantNode)
	}
}

func (app *Application) processOutboxEventsForNode(tenantNode *dragonboat.RaftNode, now time.Time) {
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
		log.Err(err).Uint64("shard_id", tenantNode.ShardID).Msg("❌ Failed to read outbox events")
		return
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		log.Err(err).Uint64("shard_id", tenantNode.ShardID).Msg("❌ Failed to decode outbox events")
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

	log.Debug().Uint64("shard_id", tenantNode.ShardID).Int("events_count", len(events)).Msg("📬 Processing outbox events")

	// Group by TenantID to avoid duplicate commands to Master if there are multiple events for same tenant
	uniqueTenants := make(map[string]bool)
	var processedEventIDs []string

	for _, event := range events {
		if event.EventType == models.EventTypeTenantActivated {
			uniqueTenants[event.TenantID] = true
		}
		processedEventIDs = append(processedEventIDs, event.ID)
	}

	// Send activation commands to Master Node
	for tenantID := range uniqueTenants {
		markActiveCmd := &tentant_command.MarkTenantActiveCommand{
			TenantID: tenantID,
		}

		fsmCmd := general_command.FSM_Command{
			Now:  utils.GetNowInInt(),
			Type: general_command.REPOSITORY_COMMAND,
			CMD:  markActiveCmd,
		}

		ctxWrite, cancelWrite := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := app.MasterNode.Write(ctxWrite, fsmCmd) // fire and forget/wait for completion, but we ignore errors here
		cancelWrite()
		if err != nil {
			log.Err(err).Str("tenant_id", tenantID).Msg("❌ Failed to mark tenant active in Master")
			return // Stop processing so we don't delete the events if we failed to relay!
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
	_, err = tenantNode.Write(ctxDel, deleteFsmCmd)
	cancelDel()
	if err != nil {
		log.Err(err).Uint64("shard_id", tenantNode.ShardID).Msg("❌ Failed to delete outbox events")
	} else {
		log.Info().Uint64("shard_id", tenantNode.ShardID).Int("deleted", len(processedEventIDs)).Msg("✅ Outbox events relayed and deleted")
	}
}
