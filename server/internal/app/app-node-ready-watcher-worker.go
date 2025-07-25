package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	auth_command "deadalus-orch/server/internal/usecase/command/auth"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartNodeReadyWatcherWorker(interval time.Duration) {
	app.NodeReadyWatcherStopper.RunWorker(func() {
		masterReadyCh := app.MasterNode.StartNodeReadyWatcher(interval)

		tenantReadyChs := make([]<-chan bool, len(app.TenantNodes))
		for i, node := range app.TenantNodes {
			tenantReadyChs[i] = node.StartNodeReadyWatcher(interval)
		}

		readyMap := make(map[int]bool) // key -1 for master, 0..N-1 for tenants
		const masterKey = -1

		defer func() {
			log.Info().Msg("🔌 Node readiness watcher stopped, ensuring Rest API is shutdown.")
			app.CloseRestAPI()
		}()

		defer func() {
			log.Info().Msg("🔌 Node readiness watcher stopped, ensuring grpc API is shutdown.")
			app.CloseGrpcAPI()
		}()

		for {
			select {
			case isReady, ok := <-masterReadyCh:
				if !ok {
					log.Warn().Msg("🛑 Master node watcher closed.")
					return
				}
				readyMap[masterKey] = isReady

			default:
				for i, ch := range tenantReadyChs {
					select {
					case ready, ok := <-ch:
						if !ok {
							log.Warn().Int("tenant", i).Msg("🛑 Tenant node watcher closed.")
							return
						}
						if !ready && app.MasterNodeIsReady {
							log.Warn().Int("tenant", i).Msg("⚠️️ Tenant node does not respond.")
						}

						if ready {
							defineColumnFamilies(app)
						}
						readyMap[i] = ready
					default:
					}
				}
			}

			allReady := readyMap[masterKey]
			for i := range tenantReadyChs {
				if !readyMap[i] {
					allReady = false
					break
				}
			}

			go func() {
				leaderID, _, valid, _ := app.MasterNode.NH.GetLeaderID(uint64(dragonboat.MasterShardID))
				if valid && leaderID == config.GlobalConfiguration.ReplicaID {
					app.MasterNodeIsLeader = true
				} else {
					app.MasterNodeIsLeader = false
				}
			}()

			if allReady && !app.MasterNodeIsReady {
				log.Info().Msg("✅ Master + all tenants ready for consensus.")
				app.MasterNodeIsReady = true

				app.StartAssignTenants()
				if dragonboat.ContainsRole(app.MasterNode.Roles, dragonboat.RoleConsensus) {
					bootstrapRootUserCmd := &auth_command.BootstrapRootUserCommand{}
					cmd := general_command.FSM_Command{
						Now:  utils.GetNowInInt(),
						Type: general_command.REPOSITORY_COMMAND,
						CMD:  bootstrapRootUserCmd,
					}

					ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
					defer cancel()
					_, err := app.MasterNode.Write(ctx, cmd)
					if err != nil {
						log.Fatal().
							Err(err).
							Str("package", "app").
							Str("func", "Run").
							Msgf("❌ Failed to bootstrap root user")
					}
				}

				if dragonboat.ContainsRole(app.MasterNode.Roles, dragonboat.RoleAdmin) {
					app.StartRestAPI()
				} else {
					app.CloseRestAPI()
				}

				if dragonboat.ContainsRole(app.MasterNode.Roles, dragonboat.RoleConnector) {
					app.StartGrpcAPI()
				} else {
					app.CloseGrpcAPI()
				}

			}

			if !allReady && app.MasterNodeIsReady {
				log.Warn().Msg("⚠️️ One or more nodes are not ready. Marking node as not ready.")
				app.MasterNodeIsReady = false
				app.CloseRestAPI()
				app.CloseGrpcAPI()
			}

			select {
			case <-app.NodeReadyWatcherStopper.ShouldStop():
				log.Info().Msg("ℹ️  NodeReadyWatcher received stop signal.")
				return
			case <-time.After(interval):
			}
		}
	})

}

func defineColumnFamilies(app *Application) {
	for _, tenantNode := range app.TenantNodes {
		for i := 0; i < config.GlobalConfiguration.MaxColumnFamilies; i++ {

			createColumnFamilyCommand := &general_command.CreateColumnFamilyCommand{
				Name:  "cf-n-" + strconv.Itoa(i),
				IsTTL: false,
			}

			ccfCmd := general_command.FSM_Command{
				Now:  utils.GetNowInInt(),
				Type: general_command.REPOSITORY_COMMAND,
				CMD:  createColumnFamilyCommand,
			}

			writeCtx, writeCancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
			defer writeCancel()

			_, err := tenantNode.Write(writeCtx, ccfCmd)
			if err != nil {
				log.Fatal().Err(err).Int("ShardID", int(tenantNode.GetClient().ShardID)).Str("ColumnFamily", createColumnFamilyCommand.Name).Msg("Failed to create column family for Shard")
			}

			createColumnFamilyCommandTtl := &general_command.CreateColumnFamilyCommand{
				Name:  "cf-ttl-" + strconv.Itoa(i),
				IsTTL: false,
			}

			ccfCmdTtl := general_command.FSM_Command{
				Now:  utils.GetNowInInt(),
				Type: general_command.REPOSITORY_COMMAND,
				CMD:  createColumnFamilyCommandTtl,
			}

			_, err = tenantNode.Write(writeCtx, ccfCmdTtl)
			if err != nil {
				log.Fatal().Err(err).Int("ShardID", int(tenantNode.GetClient().ShardID)).Str("ColumnFamily", createColumnFamilyCommandTtl.Name).Msg("Failed to create column family for Shard")
			}
		}
	}
}
