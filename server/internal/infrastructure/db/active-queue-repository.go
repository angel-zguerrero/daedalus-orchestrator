package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	models "deadalus-orch/shared/models"
)

type ActiveQueueRepository struct {
	*Repository[models.ActiveQueue]
}

func NewActiveQueueRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*ActiveQueueRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	// Note: ActiveQueue uses QueueID as its ID.
	repo, err := GetRepository[models.ActiveQueue](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &ActiveQueueRepository{Repository: repo}, nil
}

func (r *ActiveQueueRepository) PutActiveQueue(q *models.Queue, now time.Time) (*models.ActiveQueue, error) {
	aq := &models.ActiveQueue{
		ID:         q.ID,
		Code:       q.Code,
		VNamespace: q.VNamespace,
	}
	_, err := r.Create(aq, now)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			_, err = r.Update(aq, now)
			if err != nil {
				return nil, err
			}
			return aq, nil
		}
		return nil, err
	}
	return aq, nil
}

func (r *ActiveQueueRepository) DeleteActiveQueue(queueID string, now time.Time) (bool, error) {
	return r.Delete(queueID, now)
}

// PaginateWithClaimWorkFilter paginates active queues applying the DB-level rules from the ClaimWorkFilter.
// Inclusion lists, exact exclusions, and NOT LIKE pattern exclusions are evaluated in memory via a
// sequential linear scan of the active_queues data keys for extremely fast pagination.
func (r *ActiveQueueRepository) PaginateWithClaimWorkFilter(f models.ClaimWorkFilter, pageSize int, cursor string, now time.Time) (*FindResult[models.Queue], error) {
	var results []models.Queue
	currentCursorID := cursor
	def := r.GetTableDefinition()
	dataPrefix := fmt.Sprintf("%s:%s:data:", def.Schema, def.Name)

	for len(results) < pageSize {
		cursorKeyStr := ""
		if currentCursorID != "" {
			cursorKeyStr = dataPrefix + currentCursorID
		}

		// Read a chunk from pebble (fetch more than pageSize to account for filtered out items)
		items, nextCursorKeyStr, err := r.kvStore.SearchByPatternPaginatedKV(def.ColumnFamily, def.ColumnFamilySector, dataPrefix+"*", cursorKeyStr, pageSize*2, now)
		if err != nil {
			return nil, err
		}

		for _, item := range items {
			var aq models.ActiveQueue
			if err := json.Unmarshal(item.Value, &aq); err != nil {
				continue
			}
			
			// We synthesize a Queue strictly for filtering so we can reuse matchClaimWorkFilter
			q := models.Queue{
				ID:            aq.ID,
				Code:          aq.Code,
				VNamespace:    aq.VNamespace,
				MessagesCount: 1, // We know it has messages because it's in ActiveQueue
			}

			if matchClaimWorkFilter(&q, f) {
				results = append(results, q)
				if len(results) == pageSize {
					// We reached page size, current cursor should be the ID of the current item.
					parts := strings.Split(string(item.Key), ":")
					if len(parts) > 0 {
						currentCursorID = parts[len(parts)-1]
					}
					break
				}
			}
		}

		if len(results) == pageSize {
			break
		}

		if nextCursorKeyStr == "" {
			currentCursorID = ""
			break
		}
		
		// Extract ID from nextCursorKeyStr (which looks like schema:table:data:ID)
		parts := strings.Split(nextCursorKeyStr, ":")
		if len(parts) > 0 {
			currentCursorID = parts[len(parts)-1]
		}
	}

	return &FindResult[models.Queue]{
		Entities: results,
		Cursor:   currentCursorID,
	}, nil
}
