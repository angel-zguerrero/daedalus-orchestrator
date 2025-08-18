package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
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

	// Create NodeScheduler instance
	nodeScheduler := &models.NodeScheduler{
		Name: nodeSchedulerName,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Send heartbeat by calling BulkUpsertNodeScheduler
	_, err = nodeSchedulerBO.BulkUpsertNodeScheduler(ctx, []*models.NodeScheduler{nodeScheduler})
	if err != nil {
		log.Err(err).Msg("❌ Failed to send NodeScheduler heartbeat")
		return
	}

	log.Debug().Str("nodeSchedulerName", nodeSchedulerName).Msg("✅ NodeScheduler heartbeat sent successfully")
}
