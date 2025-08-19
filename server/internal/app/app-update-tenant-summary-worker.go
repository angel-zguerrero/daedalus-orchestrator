package app

import (
	"time"

	"github.com/rs/zerolog/log"
)

func (app *Application) StartTenantSummaryWorker(interval time.Duration) {
	app.TenantSummaryWorkerStopper.RunWorker(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if !app.MasterNodeIsReady {
					log.Debug().Msg("⏳ TenantSummary worker is waiting for the master node to be ready")
					continue
				}

				if !app.MasterNodeIsLeader {
					log.Debug().Msg("⏳ TenantSummary worker is waiting for the master node to be leader")
					continue
				}

				select {
				case <-app.TenantSummaryWorkerStopper.ShouldStop():
					log.Info().Msg("🛑 TenantSummary worker received stop signal before execution")
					return
				default:
				}

				go func() {
					app.updateTenantSummaries()
				}()

			case <-app.TenantSummaryWorkerStopper.ShouldStop():
				log.Info().Msg("ℹ️  TenantSummary worker stopped gracefully")
				return
			}
		}
	})
}

func (app *Application) updateTenantSummaries() {

}
