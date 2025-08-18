package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	business_logic "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartNodeSchedulerHeartbeatWorker(interval time.Duration) {
	app.NodeSchedulerHeartbeatStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !app.MasterNodeIsReady {
					log.Debug().Msg("⏳ NodeScheduler heartbeat is waiting for the master node to be ready")
					continue
				}

				if !dragonboat.ContainsRole(app.MasterNode.Roles, dragonboat.RoleScheduler) {
					continue
				}

				select {
				case <-app.NodeSchedulerHeartbeatStopper.ShouldStop():
					log.Info().Msg("🛑 NodeScheduler heartbeat received stop signal before execution")
					return
				default:
				}

				go func() {
					app.sendNodeSchedulerHeartbeat()
				}()

			case <-app.NodeSchedulerHeartbeatStopper.ShouldStop():
				log.Info().Msg("ℹ️  NodeScheduler heartbeat worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) sendNodeSchedulerHeartbeat() {
	// Get the hostname to use as the node scheduler name
	hostname, err := os.Hostname()
	if err != nil {
		log.Err(err).Msg("❌ Failed to get hostname for NodeScheduler heartbeat")
		return
	}

	// Get the process ID
	pid := os.Getpid()

	// Concatenate hostname with process ID
	nodeSchedulerName := fmt.Sprintf("%s-%d", hostname, pid)

	// Create server configuration for the business logic
	serverConfig := &common.ServerConfing{
		Logger:     log.Logger,
		MasterNode: app.MasterNode,
	}

	// Create NodeScheduler business object
	nodeSchedulerBO := business_logic.NewNodeSchedulerBO(serverConfig)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First, paginate through all existing node schedulers to update their connection status
	pageSize := 100
	cursor := ""
	allNodeSchedulers := []*models.NodeScheduler{}

	for {
		findResult, err := nodeSchedulerBO.GetNodeSchedulers(ctx, "", cursor, pageSize)
		if err != nil {
			log.Err(err).Msg("❌ Failed to paginate NodeSchedulers during heartbeat")
			break
		}

		// Convert to pointers and add to the list (without TTL and LastHeartbeat to preserve existing values)
		for _, ns := range findResult.Entities {
			nodeSchedulerCopy := ns // Create a copy to avoid reference issues
			// Don't set TTL or LastHeartbeat - let the upsert command handle these based on existing values
			allNodeSchedulers = append(allNodeSchedulers, &nodeSchedulerCopy)
		}

		// Check if we have more pages
		if findResult.Cursor == "" || len(findResult.Entities) < pageSize {
			break
		}
		cursor = findResult.Cursor
	}

	// Bulk upsert all existing node schedulers to update their connection status
	if len(allNodeSchedulers) > 0 {
		_, err = nodeSchedulerBO.BulkUpsertNodeScheduler(ctx, allNodeSchedulers)
		if err != nil {
			log.Err(err).Msg("❌ Failed to update existing NodeSchedulers connection status")
		} else {
			log.Debug().Int("count", len(allNodeSchedulers)).Msg("✅ Updated connection status for existing NodeSchedulers")
		}
	}

	// Now send heartbeat for the current server
	nodeScheduler := &models.NodeScheduler{
		Name:          nodeSchedulerName,
		LastHeartbeat: time.Now(),
		TTL:           config.GlobalConfiguration.NodeSchedulerTTL * 60, // Convert minutes to seconds
	}

	// Send heartbeat by calling BulkUpsertNodeScheduler for current server
	_, err = nodeSchedulerBO.BulkUpsertNodeScheduler(ctx, []*models.NodeScheduler{nodeScheduler})
	if err != nil {
		log.Err(err).Msg("❌ Failed to send NodeScheduler heartbeat")
		return
	}

	log.Debug().Str("nodeSchedulerName", nodeSchedulerName).Msg("✅ NodeScheduler heartbeat sent successfully")
}
