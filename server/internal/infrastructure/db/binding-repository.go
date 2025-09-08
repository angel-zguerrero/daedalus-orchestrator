package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"
)

type BindingRepository struct {
	*Repository[models.Binding]
}

func NewBindingRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*BindingRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.Binding](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &BindingRepository{Repository: repo}, nil
}

func (r *BindingRepository) CreateBinding(input *models.Binding, now time.Time) (string, error) {
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *BindingRepository) UpdateBinding(input *models.Binding, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *BindingRepository) GetBindingByCode(code string, vnamespace string, now time.Time) (*models.Binding, error) {
	query := "Code = " + code + " & VNamespace = " + vnamespace
	result, err := r.Find(query, 1, "", now)
	if err != nil {
		return nil, err
	}
	if len(result.Entities) == 0 {
		return nil, nil
	}
	return &result.Entities[0], nil
}

func (r *BindingRepository) GetBindingByExchangeAndQueue(exchangeID string, queueID string, now time.Time) (*models.Binding, error) {
	query := "ExchangeID = " + exchangeID + " & QueueID = " + queueID
	result, err := r.Find(query, 1, "", now)
	if err != nil {
		return nil, err
	}
	if len(result.Entities) == 0 {
		return nil, nil
	}
	return &result.Entities[0], nil
}

func (r *BindingRepository) GetBindingById(id string, now time.Time) (*models.Binding, error) {
	return r.FindByField("ID", id, now)
}

func (r *BindingRepository) GetBindingsByExchange(exchangeID string, now time.Time) (*FindResult[models.Binding], error) {
	query := "ExchangeID = " + exchangeID
	return r.Find(query, 100000, "", now)
}

func (r *BindingRepository) GetBindingsByQueue(queueID string, now time.Time) (*FindResult[models.Binding], error) {
	query := "QueueID = " + queueID
	return r.Find(query, 100000, "", now)
}

func (r *BindingRepository) DeleteBinding(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}

func (r *BindingRepository) Paginate(q string, pageSize int, cursor string, vNamespace string, now time.Time) (*FindResult[models.Binding], error) {
	if q == "" {
		if vNamespace == "" {
			return r.Find("ID != 0", pageSize, cursor, now) // ID != 0 Workaround
		} else {
			return r.Find("VNamespace = "+vNamespace, pageSize, cursor, now)
		}
	} else {
		if vNamespace == "" {
			return r.Find("RoutingKey LIKE *"+q+"* | Pattern LIKE *"+q+"*", pageSize, cursor, now)
		} else {
			return r.Find("(RoutingKey LIKE *"+q+"* | Pattern LIKE *"+q+"*) & VNamespace = "+vNamespace, pageSize, cursor, now)
		}
	}
}
