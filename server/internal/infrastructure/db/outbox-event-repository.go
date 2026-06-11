package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type OutboxEventRepository struct {
	repo *Repository[models.OutboxEvent]
}

func NewOutboxEventRepository(uow *UnitOfWork, factory IDGeneratorFactory) (*OutboxEventRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.OutboxEvent](uow, AdminFC, AdminFCSector, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &OutboxEventRepository{repo: repo}, nil
}

func (r *OutboxEventRepository) CreateEvent(input *models.OutboxEvent, now time.Time) (string, error) {
	input.CreatedAt = now
	return r.repo.Create(input, now)
}

func (r *OutboxEventRepository) GetAllEvents(now time.Time) ([]models.OutboxEvent, error) {
	// Paginate through all events. Since Outbox is ephemeral, it should be small.
	// But we use pagination just in case.
	var allEvents []models.OutboxEvent
	cursor := ""
	for {
		res, err := r.repo.Find("ID != 0", 100, cursor, now)
		if err != nil {
			return nil, err
		}
		allEvents = append(allEvents, res.Entities...)
		if res.Cursor == "" || len(res.Entities) == 0 {
			break
		}
		cursor = res.Cursor
	}
	return allEvents, nil
}

func (r *OutboxEventRepository) DeleteEvent(id string, now time.Time) (bool, error) {
	return r.repo.Delete(id, now)
}
