package db

import (
	"fmt"
	"time"

	"deadalus-orch/server/internal/pkg/config"
	models "deadalus-orch/shared/models"
)

type RoutingHeadersRepository struct {
	*Repository[models.RoutingHeader]
}

func NewRoutingHeadersRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*RoutingHeadersRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.RoutingHeader](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &RoutingHeadersRepository{Repository: repo}, nil
}

func (r *RoutingHeadersRepository) CreateRoutingHeader(input *models.RoutingHeader, now time.Time) (string, error) {
	if input.HeaderType == "" || input.HeaderType != models.HeaderTypeExchange && input.HeaderType != models.HeaderTypeQueue && input.HeaderType != models.HeaderTypeQueueMessage && input.HeaderType != models.HeaderTypeBinding {
		return "", fmt.Errorf("Invalid HeaderType: %s", input.HeaderType)
	}
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *RoutingHeadersRepository) UpdateRoutingHeader(input *models.RoutingHeader, now time.Time) (bool, error) {
	if input.HeaderType == "" || input.HeaderType != models.HeaderTypeExchange && input.HeaderType != models.HeaderTypeQueue && input.HeaderType != models.HeaderTypeQueueMessage && input.HeaderType != models.HeaderTypeBinding {
		return false, fmt.Errorf("Invalid HeaderType: %s", input.HeaderType)
	}
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *RoutingHeadersRepository) GetRoutingHeaderById(id string, now time.Time) (*models.RoutingHeader, error) {
	return r.FindByField("ID", id, now)
}

func (r *RoutingHeadersRepository) GetRoutingHeadersByQueue(queueID string, now time.Time) (*FindResult[models.RoutingHeader], error) {
	query := "QueueID = " + queueID
	return r.Find(query, config.GlobalConfiguration.MaxHeaders, "", now)
}

func (r *RoutingHeadersRepository) GetRoutingHeadersByExchange(exchangeID string, now time.Time) (*FindResult[models.RoutingHeader], error) {
	query := "ExchangeID = " + exchangeID
	return r.Find(query, config.GlobalConfiguration.MaxHeaders, "", now)
}

func (r *RoutingHeadersRepository) GetRoutingHeadersByMessage(messageID string, now time.Time) (*FindResult[models.RoutingHeader], error) {
	query := "MessageID = " + messageID
	return r.Find(query, config.GlobalConfiguration.MaxHeaders, "", now)
}

func (r *RoutingHeadersRepository) GetRoutingHeadersByBinding(bindingID string, now time.Time) (*FindResult[models.RoutingHeader], error) {
	query := "BindingID = " + bindingID
	return r.Find(query, config.GlobalConfiguration.MaxHeaders, "", now)
}

func (r *RoutingHeadersRepository) DeleteRoutingHeader(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}
