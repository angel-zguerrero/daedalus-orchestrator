package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	business_logic "deadalus-orch/server/internal/usecase/business-logic"
	"deadalus-orch/shared/models"
	"time"

	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func (app *Application) StartNodeSchedulerBalancingWorker(interval time.Duration) {
	app.NodeSchedulerBalancingStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		isFirstExecution := true

		for {
			select {
			case <-ticker.C:
				if !app.MasterNodeIsReady {
					log.Debug().Msg("⏳ NodeScheduler balancing worker is waiting for the master node to be ready")
					continue
				}

				if !app.MasterNodeIsLeader {
					// Only the leader processes the balancing state
					continue
				}

				select {
				case <-app.NodeSchedulerBalancingStopper.ShouldStop():
					log.Info().Msg("🛑 NodeScheduler balancing worker received stop signal before execution")
					return
				default:
				}

				app.checkAndBalanceNodeSchedulers(isFirstExecution)
				isFirstExecution = false

			case <-app.NodeSchedulerBalancingStopper.ShouldStop():
				log.Info().Msg("ℹ️  NodeScheduler balancing worker stopped gracefully")
				return
			}
		}
	})
}

// generateUUID returns a new UUID string without dashes
func generateUUID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func (app *Application) checkAndBalanceNodeSchedulers(isFirstExecution bool) {
	serverConfig := &common.ServerConfing{
		Logger:     log.Logger,
		MasterNode: app.MasterNode,
	}

	balancingBO := business_logic.NewNodeSchedulerBalancingBO(serverConfig)
	nodeSchedulerBO := business_logic.NewNodeSchedulerBO(serverConfig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Get current state
	state, err := balancingBO.GetState(ctx)
	if err != nil {
		log.Err(err).Msg("❌ Failed to get node scheduler balancing state")
		return
	}

	// 2. Initialize if it doesn't exist
	if state == nil || state.ID == "" {
		log.Info().Msg("🆕 Node scheduler balancing state not found, creating initial state 'waiting-for-node-schedulers'")
		initialState := models.NodeSchedulerBalancingState{
			ID:          models.NodeSchedulerBalancingStateID,
			BalancingId: generateUUID(),
			Status:      models.WaitingForNodeSchedulers,
		}
		err := balancingBO.UpsertState(ctx, initialState)
		if err != nil {
			log.Err(err).Msg("❌ Failed to initialize node scheduler balancing state")
			return
		}
		state = &initialState
	} else if isFirstExecution {
		// If it's the first execution and state already exists, we reset the status to waiting
		log.Info().Msg("🚀 First execution after startup. Resetting balancing state status to 'waiting-for-node-schedulers'")
		state.Status = models.WaitingForNodeSchedulers
		state.LastNodeSchedulerIndices = make(map[int]int)
		state.BalancingId = generateUUID()
		err := balancingBO.UpsertState(ctx, *state)
		if err != nil {
			log.Err(err).Msg("❌ Failed to reset node scheduler balancing state status on startup")
			return
		}
	}

	// 3. If balanced, we periodically assign unassigned queues
	if state.Status == models.Balanced {
		log.Debug().Msg("⚖️ System is balanced. Checking for unassigned queues...")
		lastIndices, err := balancingBO.BalanceNodeSchedulers(ctx, app.TenantNodes, models.Unsupervised, state.LastNodeSchedulerIndices, state.BalancingId)
		if err != nil {
			log.Err(err).Msg("❌ Failed to perform partial node scheduler balancing")
			return
		}

		// Update state with updated indices if they changed
		state.LastNodeSchedulerIndices = lastIndices
		err = balancingBO.UpsertState(ctx, *state)
		if err != nil {
			log.Err(err).Msg("❌ Failed to update node scheduler balancing indices")
		}
		return
	}

	// 3.1. If rebalancing is requested, transition to waiting
	if state.Status == models.RequestForNewBalancing {
		log.Info().Msg("🔄 Rebalancing requested. Transitioning to 'waiting-for-node-schedulers' to stabilize")

		// Check if ALL connected node schedulers are stopped (regardless of BalancingId)
		// We need to check ALL because new node schedulers might have different BalancingId
		pageSize := 100
		cursor := ""
		allAreStopped := true

		for {
			findResult, err := nodeSchedulerBO.GetNodeSchedulers(ctx, "", "", models.ConnectionStatusConnected, -1, cursor, pageSize)
			if err != nil {
				log.Err(err).Msg("❌ Failed to get node schedulers for rebalancing check")
				return
			}

			for _, ns := range findResult.Entities {
				if ns.RunningStatus != models.NodeSchedulerRunningStatusStopped && ns.RunningStatus != "" {
					allAreStopped = false
					log.Debug().Str("nodeSchedulerName", ns.Name).Str("runningStatus", string(ns.RunningStatus)).Str("balancingId", ns.BalancingId).Msg("⏳ Node scheduler not stopped yet")
					break
				}
			}

			if !allAreStopped || findResult.Cursor == "" || len(findResult.Entities) < pageSize {
				break
			}
			cursor = findResult.Cursor
		}

		if allAreStopped {
			log.Info().Msg("✅ All node schedulers are stopped. Transitioning to 'waiting-for-node-schedulers'")
			state.Status = models.WaitingForNodeSchedulers
			state.LastNodeSchedulerIndices = make(map[int]int) // Reset indices for new balancing cycle
			state.BalancingId = generateUUID()                 // Generate new balancing ID for synchronization
			log.Info().Str("newBalancingId", state.BalancingId).Msg("🆔 Generated new BalancingId for rebalancing cycle")
			err = balancingBO.UpsertState(ctx, *state)
			if err != nil {
				log.Err(err).Msg("❌ Failed to update node scheduler balancing state to 'waiting-for-node-schedulers'")
				return
			}
		} else {
			log.Debug().Msg("⏳ Waiting for all node schedulers to stop...")
			return
		}
	}

	// 4. If waiting, check node schedulers
	if state.Status == models.WaitingForNodeSchedulers {
		// Get ALL connected node schedulers to find the latest created one
		// Note: We don't filter by BalancingId here because node schedulers need time to sync via heartbeats
		pageSize := 100
		cursor := ""
		totalFetched := 0
		var latestCreated time.Time

		for {
			findResult, err := nodeSchedulerBO.GetNodeSchedulers(ctx, "", "", models.ConnectionStatusConnected, -1, cursor, pageSize)
			if err != nil {
				log.Err(err).Msg("❌ Failed to get node schedulers for balancing check")
				return
			}

			totalFetched += len(findResult.Entities)
			for _, ns := range findResult.Entities {
				if ns.CreatedAt.After(latestCreated) {
					latestCreated = ns.CreatedAt
				}
			}

			if findResult.Cursor == "" || len(findResult.Entities) < pageSize {
				break
			}
			cursor = findResult.Cursor
		}

		log.Debug().Msgf("⏳ Total fetched node schedulers: %d", totalFetched)

		if totalFetched == 0 {
			log.Debug().Msgf("⏳ No node schedulers found yet, waiting...")
			return
		}

		// 5. Check if wait time has passed
		waitTime := config.GlobalConfiguration.NodeSchedulerBalancingWaitTime
		if time.Since(latestCreated) > waitTime {
			// 5.1. Before balancing, verify that node schedulers have synchronized their BalancingId
			// This prevents balancing with incomplete data
			syncedSchedulers := 0
			cursor = ""
			for {
				findResult, err := nodeSchedulerBO.GetNodeSchedulers(ctx, "", state.BalancingId, models.ConnectionStatusConnected, -1, cursor, pageSize)
				if err != nil {
					log.Err(err).Msg("❌ Failed to get node schedulers with current BalancingId")
					return
				}

				syncedSchedulers += len(findResult.Entities)

				if findResult.Cursor == "" || len(findResult.Entities) < pageSize {
					break
				}
				cursor = findResult.Cursor
			}

			log.Debug().Int("syncedSchedulers", syncedSchedulers).Int("totalSchedulers", totalFetched).Str("balancingId", state.BalancingId).Msg("📊 BalancingId synchronization status")

			// Only proceed if we have a reasonable number of synchronized schedulers
			if syncedSchedulers > 0 && syncedSchedulers >= totalFetched/2 { // At least 50% synchronized
				log.Info().Msg("⚖️  Wait time passed and BalancingId synchronized. Starting balancing...")

				lastIndices, err := balancingBO.BalanceNodeSchedulers(ctx, app.TenantNodes, "", state.LastNodeSchedulerIndices, state.BalancingId)
				if err != nil {
					log.Err(err).Msg("❌ Failed to balance node schedulers")
					return
				}

				// 6. Update state to balanced
				state.Status = models.Balanced
				state.LastNodeSchedulerIndices = lastIndices
				err = balancingBO.UpsertState(ctx, *state)
				if err != nil {
					log.Err(err).Msg("❌ Failed to update node scheduler balancing state to 'balanced'")
					return
				}
				log.Info().Msg("✅ Node schedulers balanced successfully")
			} else {
				log.Debug().Msg("⏳ Waiting for more node schedulers to synchronize BalancingId via heartbeats...")
			}
		} else {
			remaining := waitTime - time.Since(latestCreated)
			log.Debug().Interface("remaining", remaining).Msg("⏳ Waiting for node scheduler creations to stabilize")
		}
	}
}
