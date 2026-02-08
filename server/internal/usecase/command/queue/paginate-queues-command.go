package queue

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"
)

func init() {
	gob.Register(PaginateQueuesCommand{})
	gob.Register(db.FindResult[models.Queue]{})
	gob.Register(models.RoutingHeader{})
}

type PaginateQueuesCommand struct {
	Query          string
	Cursor         string
	PageSize       int
	VNamespace     string
	IncludeHeaders bool
	CF             string
	CFS            string
}

func (cmd *PaginateQueuesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	findResult, err := queueRepo.Paginate(cmd.Query, cmd.PageSize, cmd.Cursor, cmd.VNamespace, now)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// If headers are requested, populate them using RoutingHeader
	if cmd.IncludeHeaders && findResult.Entities != nil {
		routingHeadersRepo, err := db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		// Populate headers for each queue
		for i := range findResult.Entities {
			queue := &findResult.Entities[i]

			// Get headers for this queue using QueueID
			if headersResult, err := routingHeadersRepo.GetRoutingHeadersByQueue(queue.ID, now); err == nil && headersResult != nil {
				headers := make(map[string]string)
				for _, header := range headersResult.Entities {
					headers[header.Key] = header.Value
				}
				queue.Headers = headers
			}
		}
	}

	// Populate NodeScheduler supervisor fields
	if len(findResult.Entities) > 0 {
		// Collect unique NodeSchedulerSupervisorIds
		nodeSchedulerIds := make(map[string]bool)
		for _, queue := range findResult.Entities {
			if queue.NodeSchedulerSupervisorId != "" {
				nodeSchedulerIds[queue.NodeSchedulerSupervisorId] = true
			}
		}

		// If there are NodeSchedulerSupervisorIds, fetch the NodeSchedulers
		if len(nodeSchedulerIds) > 0 {
			nodeSchedulerRepo, err := db.NewNodeSchedulerRepository(uow, idFactory)
			if err == nil {
				// Create a map of NodeScheduler ID -> NodeScheduler
				nodeSchedulerMap := make(map[string]*models.NodeScheduler)
				for id := range nodeSchedulerIds {
					if ns, err := nodeSchedulerRepo.GetNodeSchedulerById(id, now); err == nil && ns != nil {
						nodeSchedulerMap[id] = ns
					}
				}

				// Populate the virtual fields
				for i := range findResult.Entities {
					queue := &findResult.Entities[i]
					if queue.NodeSchedulerSupervisorId != "" {
						if ns, exists := nodeSchedulerMap[queue.NodeSchedulerSupervisorId]; exists {
							queue.NodeSchedulerSupervisorCode = ns.ID
							queue.NodeSchedulerSupervisorName = ns.Name
						}
					}
				}
			}
		}
	}

	commandResult.Result = *findResult
	return *commandResult
}
