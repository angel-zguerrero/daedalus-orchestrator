package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
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
		cmd := commands.FSM_Command{
			Now:  utils.GetNowInInt(),
			Type: commands.MCL,
			CMD: commands.MCLK_Command{
				Op: commands.ClearExpiredTTL,
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
		defer cancel()

		if _, err := app.MasterNode.Write(ctx, cmd); err != nil {
			log.Err(err).Msg("❌ Failed to clear TTL on master node")
		} else {
			log.Info().Msg("✅ TTL cleared on master node")
		}
	}
}

func (app *Application) clearTenantBatchTTL(batch []*dragonboat.RaftNode) {
	for _, node := range batch {
		cmd := commands.FSM_Command{
			Now:  utils.GetNowInInt(),
			Type: commands.MCL,
			CMD: commands.MCLK_Command{
				Op: commands.ClearExpiredTTL,
			},
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
		defer cancel()

		if _, err := node.Write(ctx, cmd); err != nil {
			log.Err(err).
				Str("node", strconv.FormatUint(node.ShardID, 10)).
				Msg("❌ Failed to clear TTL on tenant node")
		} else {
			log.Info().
				Str("node", strconv.FormatUint(node.ShardID, 10)).
				Msg("✅ TTL cleared on tenant node")
		}
	}
}
