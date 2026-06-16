package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(GetQueueGaugesCommand{})
	gob.Register([]models.QueueGauges{})
}

type GetQueueGaugesCommand struct {
	CF  string
	CFS string
}

func (cmd *GetQueueGaugesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	var allGauges []models.QueueGauges
	cursor := ""
	pageSize := 1000 // Large page size to minimize iterations

	for {
		// Paginate all queues, we use ID != 0 to get all queues
		findResult, err := queueRepo.Paginate("", pageSize, cursor, "", now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		for _, q := range findResult.Entities {
			// We only care about active or draining queues
			if q.State == models.QueueActive || q.State == models.QueueDraining {
				allGauges = append(allGauges, models.QueueGauges{
					QueueCode:  q.Code,
					VNamespace: q.VNamespace,
					Pending:    uint64(q.MessagesCount),
					InProcess:  uint64(q.CurrentDeliveringMessages),
				})
			}
		}

		if findResult.Cursor == "" || len(findResult.Entities) < pageSize {
			break
		}
		cursor = findResult.Cursor
	}

	commandResult.Result = allGauges
	return *commandResult
}
