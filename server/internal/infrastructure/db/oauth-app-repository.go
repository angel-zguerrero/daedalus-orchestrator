package db

import (
	"deadalus-orch/shared/models"
	"time"
)

type OAuthAppRepository struct {
	repo *Repository[models.OAuthApp]
}

func NewOAuthAppRepository(uow *UnitOfWork, idFactory IDGeneratorFactory) (*OAuthAppRepository, error) {
	repo, err := GetRepository[models.OAuthApp](uow, MasterEventFC, MasterEventFCSector, "oauth_apps_schema", idFactory)
	if err != nil {
		return nil, err
	}
	return &OAuthAppRepository{repo: repo}, nil
}

func (r *OAuthAppRepository) Save(app *models.OAuthApp, now time.Time) error {
	if app.ID == "" {
		_, err := r.repo.Create(app, now)
		return err
	}
	_, err := r.repo.Update(app, now)
	return err
}

func (r *OAuthAppRepository) FindByID(id string, now time.Time) (*models.OAuthApp, error) {
	result, err := r.repo.Find("ID="+id, 1, "", now)
	if err != nil {
		return nil, err
	}
	if result != nil && len(result.Entities) > 0 {
		app := result.Entities[0]
		return &app, nil
	}
	return nil, nil
}

func (r *OAuthAppRepository) FindByClientID(clientID string, now time.Time) (*models.OAuthApp, error) {
	app, err := r.repo.FindByField("ClientID", clientID, now)
	if err != nil {
		return nil, err
	}
	if app != nil {
		return app, nil
	}
	return nil, nil
}

func (r *OAuthAppRepository) Delete(id string, now time.Time) error {
	_, err := r.repo.Delete(id, now)
	return err
}

func (r *OAuthAppRepository) FindAllByTenantID(tenantID string, now time.Time) ([]models.OAuthApp, error) {
	result, err := r.repo.Find("TenantID="+tenantID, -1, "", now)
	if err != nil {
		return nil, err
	}
	if result != nil {
		return result.Entities, nil
	}
	return []models.OAuthApp{}, nil
}
