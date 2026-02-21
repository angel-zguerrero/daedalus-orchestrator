package app

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/server/common"
	business_logic "deadalus-orch/server/internal/usecase/business-logic"
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartJobWorkerHeartbeatMonitor(interval time.Duration) {
	app.JobWorkerHeartbeatStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !app.MasterNodeIsReady {
					continue
				}

				if !app.MasterNodeIsLeader {
					continue
				}

				select {
				case <-app.JobWorkerHeartbeatStopper.ShouldStop():
					log.Info().Msg("🛑 JobWorker heartbeat monitor received stop signal before execution")
					return
				default:
				}

				app.reviewJobWorkersHeartbeat()

			case <-app.JobWorkerHeartbeatStopper.ShouldStop():
				log.Info().Msg("ℹ️  JobWorker heartbeat monitor worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) reviewJobWorkersHeartbeat() {
	// Use a background context with timeout for the whole review process
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	jobWorkerBO := app.getJobWorkerBO()
	jobWorkerBO.ReviewJobWorkersConnectionStatus(ctx)
}

func (app *Application) getJobWorkerBO() *business_logic.JobWorkerBO {
	return business_logic.NewJobWorkerBO(app.getServerConfig())
}

func (app *Application) getServerConfig() *common.ServerConfing {
	return &common.ServerConfing{
		Logger:     log.Logger,
		MasterNode: app.MasterNode,
	}
}
