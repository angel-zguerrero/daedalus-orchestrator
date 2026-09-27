package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type RoutingHeadersRepository struct {
	*Repository[models.RoutingHeader]
}

func NewRoutingHeadersRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*RoutingHeadersRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.RoutingHeader](uow, cf, cfs, "routing_headers_schema", factory)
	if err != nil {
		return nil, err
	}
	return &RoutingHeadersRepository{Repository: repo}, nil
}

func (r *RoutingHeadersRepository) CreateRoutingHeader(input *models.RoutingHeader, now time.Time) (string, error) {
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *RoutingHeadersRepository) DeleteRoutingHeader(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}

func (r *RoutingHeadersRepository) UpdateRoutingHeader(input *models.RoutingHeader, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *RoutingHeadersRepository) GetRoutingHeadersByQueue(queueID string, now time.Time) (*FindResult[models.RoutingHeader], error) {
	query := fmt.Sprintf("QueueID = %s & HeaderType = %s", queueID, models.HeaderTypeQueue)
	return r.Find(query, 1000, "", now)
}

func (r *RoutingHeadersRepository) GetRoutingHeadersByQueueMessage(messageID string, now time.Time) (*FindResult[models.RoutingHeader], error) {
	query := fmt.Sprintf("QueueMessageID = %s & HeaderType = %s", messageID, models.HeaderTypeQueueMessage)
	return r.Find(query, 1000, "", now)
}

func (r *RoutingHeadersRepository) GetRoutingHeadersByMessage(messageID string, now time.Time) (*FindResult[models.RoutingHeader], error) {
	return r.GetRoutingHeadersByQueueMessage(messageID, now)
}
