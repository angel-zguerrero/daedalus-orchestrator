package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(EnqueueCommand{})
	gob.Register(models.QueuePartition{})
	gob.Register([]models.QueuePartition{})
	gob.Register(models.QueueMessage{})
	gob.Register([]models.QueueMessage{})
}

type EnqueueCommand struct {
}

func (cmd *EnqueueCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	return *commandResult
}
