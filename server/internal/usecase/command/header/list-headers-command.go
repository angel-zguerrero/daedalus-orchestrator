package header_command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(ListHeadersCommand{})
	gob.Register(db.FindResult[models.RoutingHeader]{})
}

type ListHeadersCommand struct {
	Key               string
	RoutingHeaderType models.RoutingHeaderType
	VNamespace        string
	CF                string
	CFS               string
}

func (cmd *ListHeadersCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	idFactory := &db.DeterministicIDGeneratorFactory{}
	routingHeadersRepo, err := db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	// Collect all results by paginating with cursor internally
	var allHeaders []models.RoutingHeader
	cursor := "" // Start with empty cursor

	for {
		var findResult *db.FindResult[models.RoutingHeader]
		var err error

		// Use different repository methods based on the header type
		switch cmd.RoutingHeaderType {
		case models.HeaderTypeQueueMessage:
			// For queue messages, construct query and use generic Find method
			query := "HeaderType = " + string(models.HeaderTypeQueueMessage) + " & Key = " + cmd.Key
			findResult, err = routingHeadersRepo.Find(query, config.GlobalConfiguration.MaxHeaders, cursor, now)
		case models.HeaderTypeQueue:
			// For queue headers, construct query and use generic Find method
			query := "HeaderType = " + string(models.HeaderTypeQueue) + " & Key = " + cmd.Key
			findResult, err = routingHeadersRepo.Find(query, config.GlobalConfiguration.MaxHeaders, cursor, now)
		case models.HeaderTypeExchange:
			// For exchange headers, construct query and use generic Find method
			query := "HeaderType = " + string(models.HeaderTypeExchange) + " & Key = " + cmd.Key
			findResult, err = routingHeadersRepo.Find(query, config.GlobalConfiguration.MaxHeaders, cursor, now)
		case models.HeaderTypeBinding:
			// For binding headers, construct query and use generic Find method
			query := "HeaderType = " + string(models.HeaderTypeBinding) + " & Key = " + cmd.Key
			findResult, err = routingHeadersRepo.Find(query, config.GlobalConfiguration.MaxHeaders, cursor, now)
		default:
			commandResult.Error = fmt.Sprintf("unsupported routing header type: %s", cmd.RoutingHeaderType)
			return *commandResult
		}

		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		// Add this page's results to our collection
		allHeaders = append(allHeaders, findResult.Entities...)

		// If the cursor is empty, it means there are no more results
		if findResult.Cursor == "" {
			break
		}

		// Update cursor for next iteration
		cursor = findResult.Cursor
	}

	commandResult.Result = allHeaders
	return *commandResult
}
