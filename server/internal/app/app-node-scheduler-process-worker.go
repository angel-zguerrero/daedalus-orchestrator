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

func (app *Application) StartNodeSchedulerHeartbeat(interval time.Duration) {
	for i := range app.TenantNodes {
		index := i
		app.NodeSchedulerProcessStopper.RunWorker(func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if !app.MasterNodeIsReady {
						//log.Debug().Int("index", index).Msg("⏳ NodeScheduler process is waiting for the master node to be ready")
						continue
					}

					if !dragonboat.ContainsRole(app.MasterNode.Roles, dragonboat.RoleScheduler) {
						continue
					}

					select {
					case <-app.NodeSchedulerProcessStopper.ShouldStop():
						log.Info().Int("index", index).Msg("🛑 NodeScheduler process received stop signal before execution")
						return
					default:
					}

					app.sendNodeSchedulerHeartbeat(index)

				case <-app.NodeSchedulerProcessStopper.ShouldStop():
					log.Info().Int("index", index).Msg("ℹ️  NodeScheduler process worker stopped gracefully")
					return
				}
			}
		})
	}
}

func (app *Application) StartNodeSchedulerProcessWorkers(interval time.Duration) {
	// Worker for checking connection status (runs only on Leader)
	app.NodeSchedulerProcessStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !app.MasterNodeIsReady {
					continue
				}

				if app.MasterNodeIsLeader {
					app.reviewNodeSchedulersConnectionStatus()
				}
			case <-app.NodeSchedulerProcessStopper.ShouldStop():
				return
			}
		}
	})

	for i := range app.TenantNodes {
		index := i
		app.NodeSchedulerProcessStopper.RunWorker(func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if !app.MasterNodeIsReady {
						continue
					}

					if !dragonboat.ContainsRole(app.MasterNode.Roles, dragonboat.RoleScheduler) {
						continue
					}

					select {
					case <-app.NodeSchedulerProcessStopper.ShouldStop():
						log.Info().Int("index", index).Msg("🛑 NodeScheduler process received stop signal before execution")
						return
					default:
					}

					app.processNodeSchedulerTasks(index)

				case <-app.NodeSchedulerProcessStopper.ShouldStop():
					log.Info().Int("index", index).Msg("ℹ️  NodeScheduler process worker stopped gracefully")
					return
				}
			}
		})
	}
}

func (app *Application) processNodeSchedulerTasks(tenantNodeIndex int) {

	// Get the hostname to use as the node scheduler name
	hostname, err := os.Hostname()
	if err != nil {
		log.Err(err).Msg("❌ Failed to get hostname for NodeScheduler heartbeat")
		return
	}

	// Get the process ID
	pid := os.Getpid()

	// Concatenate hostname with process ID and index
	nodeSchedulerName := fmt.Sprintf("%s-%d-%d", hostname, pid, tenantNodeIndex)

	// Create server configuration for the business logic
	serverConfig := &common.ServerConfing{
		Logger:     log.Logger,
		MasterNode: app.MasterNode,
	}

	nodeSchedulerBO := business_logic.NewNodeSchedulerBO(serverConfig)

	getCtx, getCancel := context.WithTimeout(context.Background(), 10*time.Second)
	nodeScheduler, err := nodeSchedulerBO.GetNodeSchedulerByName(getCtx, nodeSchedulerName)
	getCancel()
	if err != nil {
		log.Err(err).Msg("❌ Failed to get node scheduler during process node scheduler tasks " + nodeSchedulerName)
	}

	if nodeScheduler.RunningStatus == models.NodeSchedulerRunningStatusRunning {
		balancingBO := business_logic.NewNodeSchedulerBalancingBO(serverConfig)

		stateCtx, stateCancel := context.WithTimeout(context.Background(), 10*time.Second)
		state, err := balancingBO.GetState(stateCtx)
		stateCancel()
		if err != nil {
			log.Err(err).Msg("❌ Failed to get node scheduler balancing state")
			return
		}
		if state != nil && state.Status == models.RequestForNewBalancing {
			fmt.Println("🔄 Starting new balancing process for node scheduler " + nodeSchedulerName)
			updateCtx, updateCancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, err = nodeSchedulerBO.UpdateRunningStatusNodeScheduler(updateCtx, []*models.NodeScheduler{&nodeScheduler}, models.NodeSchedulerRunningStatusStopped)
			updateCancel()
			if err != nil {
				log.Err(err).Msg("❌ Error updating running status node schedulers")
			}
		} else if state != nil && state.Status == models.Balanced {
			//log.Debug().Any("Node scheduler", nodeScheduler).Msg("🔍 Reviewing node scheduler tasks (placeholder)")
		}
	}

}

func (app *Application) reviewNodeSchedulersConnectionStatus() {

	serverConfig := &common.ServerConfing{
		Logger:     log.Logger,
		MasterNode: app.MasterNode,
	}

	// Create NodeScheduler business object
	nodeSchedulerBO := business_logic.NewNodeSchedulerBO(serverConfig)
	// First, paginate through all existing node schedulers to update their connection status
	pageSize := 100
	cursor := ""
	allNodeSchedulers := []*models.NodeScheduler{}

	for {
		paginateCtx, paginateCancel := context.WithTimeout(context.Background(), 10*time.Second)
		findResult, err := nodeSchedulerBO.GetNodeSchedulers(paginateCtx, "", "", models.ConnectionStatusConnected, -1, cursor, pageSize)
		paginateCancel()
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
		// Don't modify BalancingId - preserve the existing value to maintain synchronization
		// The BalancingId is managed by the balancing system and heartbeat process
		upsertCtx, upsertCancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := nodeSchedulerBO.BulkUpsertNodeScheduler(upsertCtx, allNodeSchedulers)
		upsertCancel()
		if err != nil {
			log.Err(err).Msg("❌ Failed to update existing NodeSchedulers connection status")
		} else {
			log.Debug().Int("count", len(allNodeSchedulers)).Msg("✅ Updated connection status for existing NodeSchedulers")
		}
	}

}

func (app *Application) sendNodeSchedulerHeartbeat(tenantNodeIndex int) {
	// Get the hostname to use as the node scheduler name
	hostname, err := os.Hostname()
	if err != nil {
		log.Err(err).Msg("❌ Failed to get hostname for NodeScheduler heartbeat")
		return
	}

	// Get the process ID
	pid := os.Getpid()

	// Concatenate hostname with process ID and index
	nodeSchedulerName := fmt.Sprintf("%s-%d-%d", hostname, pid, tenantNodeIndex)

	// Create server configuration for the business logic
	serverConfig := &common.ServerConfing{
		Logger:     log.Logger,
		MasterNode: app.MasterNode,
	}

	// Create NodeScheduler business object
	nodeSchedulerBO := business_logic.NewNodeSchedulerBO(serverConfig)

	// Each Raft operation gets its own fresh context to avoid "invalid deadline" errors.
	// A single shared context would cause later operations to fail if earlier ones are slow
	// (e.g., during cluster reconfiguration when adding new members).
	balancingBO := business_logic.NewNodeSchedulerBalancingBO(serverConfig)

	getStateCtx, getStateCancel := context.WithTimeout(context.Background(), 10*time.Second)
	state, err := balancingBO.GetState(getStateCtx)
	getStateCancel()
	if err != nil {
		log.Err(err).Msg("❌ Failed to get node scheduler balancing state")
		return
	}

	// Now send heartbeat for the current server
	nodeScheduler := &models.NodeScheduler{
		Name:                    nodeSchedulerName,
		LastHeartbeat:           time.Now(),
		TTL:                     config.GlobalConfiguration.NodeSchedulerTTL * 60, // Convert minutes to seconds
		AssignedTenantNodeIndex: tenantNodeIndex,
		BalancingId:             state.BalancingId, // update the balancing id
	}

	// Send heartbeat by calling BulkUpsertNodeScheduler for current server
	heartbeatCtx, heartbeatCancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, err = nodeSchedulerBO.BulkUpsertNodeScheduler(heartbeatCtx, []*models.NodeScheduler{nodeScheduler})
	heartbeatCancel()
	if err != nil {
		log.Err(err).Msg("❌ Failed to send NodeScheduler heartbeat")
		return
	}

	//log.Debug().Str("nodeSchedulerName", nodeSchedulerName).Msg("✅ NodeScheduler heartbeat sent successfully")
}
