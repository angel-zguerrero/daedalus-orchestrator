package db

import (
	"fmt"
	"strings"
	"time"

	models "deadalus-orch/shared/models"
)

type ExchangeRepository struct {
	*Repository[models.Exchange]
}

func NewExchangeRepository(uow *UnitOfWork, factory IDGeneratorFactory, cf, cfs string) (*ExchangeRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.Exchange](uow, cf, cfs, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &ExchangeRepository{Repository: repo}, nil
}

func (r *ExchangeRepository) CreateExchange(input *models.Exchange, now time.Time) (string, error) {
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *ExchangeRepository) UpdateExchange(input *models.Exchange, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *ExchangeRepository) GetExchangeByName(name string, vNamespace string, now time.Time) (*models.Exchange, error) {
	result, err := r.Find("Name="+name+" & VNamespace="+vNamespace, 1, "", now)
	if err != nil {
		return nil, err
	}
	if len(result.Entities) == 0 {
		return nil, nil
	}
	return &result.Entities[0], nil
}
func (r *ExchangeRepository) GetExchangeById(id string, now time.Time) (*models.Exchange, error) {
	return r.FindByField("ID", id, now)
}

func (r *ExchangeRepository) Paginate(q string, pageSize int, cursor string, vNamespace string, now time.Time) (*FindResult[models.Exchange], error) {
	var query string

	if q == "" && vNamespace == "" {
		query = "ID != 0" // ID != 0 Workaround
	} else {
		var conditions []string

		// Add name search condition if q is provided
		if q != "" {
			conditions = append(conditions, "Name LIKE *"+q+"*")
		}

		// Add vNamespace filter condition if vNamespace is provided
		if vNamespace != "" {
			conditions = append(conditions, "VNamespace LIKE *"+vNamespace+"*")
		}

		// If no conditions but we got here, use the workaround
		if len(conditions) == 0 {
			query = "ID != 0"
		} else {
			query = strings.Join(conditions, " AND ")
		}
	}

	return r.Find(query, pageSize, cursor, now)
}

func (r *ExchangeRepository) DeleteExchangeByName(name string, now time.Time) (bool, error) {
	rootExchange, err := r.FindByField("Name", name, now)
	if err != nil || rootExchange == nil {
		return false, err
	}
	if err != nil {
		return false, err
	}

	return r.Delete(rootExchange.ID, now)
}

func (r *ExchangeRepository) DeleteExchangeById(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}
