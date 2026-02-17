package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/utils"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartNodeClearExpiredTTLWorker(interval time.Duration, batchSize int) {
	var cleaningLock sync.Mutex

	app.NodeClearExpiredTTLStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !cleaningLock.TryLock() {
					log.Warn().Msg("⏳ Skipping TTL cleanup: previous execution still in progress")
					continue
				}

				go func() {
					defer cleaningLock.Unlock()

					if !app.MasterNodeIsReady {
						log.Warn().Msg("⏳ TTL cleaner is waiting for the master node to be ready")
						return
					}

					if !app.MasterNodeIsLeader {
						log.Warn().Msg("only leader can delete ttl keys")
						return
					}

					select {
					case <-app.NodeClearExpiredTTLStopper.ShouldStop():
						log.Info().Msg("🛑 TTL cleaner received stop signal before starting")
						return
					default:
					}

					app.clearMasterNodeTTL()

					tenantCount := len(app.TenantNodes)
					for i := 0; i < tenantCount; i += batchSize {
						select {
						case <-app.NodeClearExpiredTTLStopper.ShouldStop():
							log.Warn().Msg("🛑 TTL cleaner interrupted during tenant batch execution")
							return
						default:
						}

						end := i + batchSize
						if end > tenantCount {
							end = tenantCount
						}

						batch := app.TenantNodes[i:end]
						app.clearTenantBatchTTL(batch)
					}
				}()

			case <-app.NodeClearExpiredTTLStopper.ShouldStop():
				log.Info().Msg("ℹ️  TTL cleaner worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) clearMasterNodeTTL() {
	if app.MasterNodeIsReady && dragonboat.ContainsRole(app.MasterNode.Roles, dragonboat.RoleConsensus) {
		cmd := general_command.FSM_Command{
			Now:  utils.GetNowInInt(),
			Type: general_command.MCL,
			CMD: general_command.MCLK_Command{
				Op: general_command.ClearExpiredTTL,
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
		defer cancel()

		resultChan, err := app.MasterNode.Write(ctx, cmd)
		if err != nil {
			log.Err(err).Msg("❌ Failed to start TTL clear operation on master node")
			return
		}

		go func() {
			select {
			case writeResult := <-resultChan:
				if writeResult.Error != nil {
					log.Err(writeResult.Error).Msg("❌ Failed to clear TTL on master node")
				} else {
					log.Info().Msg("✅ TTL cleared on master node")
				}
			case <-ctx.Done():
				log.Warn().Msg("⏱️ TTL clear operation timed out on master node")
			}
		}()
	}
}

func (app *Application) clearTenantBatchTTL(batch []*dragonboat.RaftNode) {
	for _, node := range batch {
		cmd := general_command.FSM_Command{
			Now:  utils.GetNowInInt(),
			Type: general_command.MCL,
			CMD: general_command.MCLK_Command{
				Op: general_command.ClearExpiredTTL,
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
		defer cancel()

		resultChan, err := node.Write(ctx, cmd)
		if err != nil {
			log.Err(err).
				Str("node", strconv.FormatUint(node.ShardID, 10)).
				Msg("❌ Failed to start TTL clear operation on tenant node")
			return
		}

		go func(nodeID uint64) {
			select {
			case writeResult := <-resultChan:
				if writeResult.Error != nil {
					log.Err(writeResult.Error).
						Str("node", strconv.FormatUint(nodeID, 10)).
						Msg("❌ Failed to clear TTL on tenant node")
				} else {
					log.Info().
						Str("node", strconv.FormatUint(nodeID, 10)).
						Msg("✅ TTL cleared on tenant node")
				}
			case <-ctx.Done():
				log.Warn().
					Str("node", strconv.FormatUint(nodeID, 10)).
					Msg("⏱️ TTL clear operation timed out on tenant node")
			}
		}(node.ShardID)
	}
}
