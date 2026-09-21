package db

import (
	"deadalus-orch/shared/models"
	"time"
)

type OAuthTokenRepository struct {
	repo *Repository[models.OAuthToken]
}

func NewOAuthTokenRepository(uow *UnitOfWork, idFactory IDGeneratorFactory) (*OAuthTokenRepository, error) {
	repo, err := GetRepository[models.OAuthToken](uow, MasterEventFC, MasterEventFCSector, "oauth_tokens_schema", idFactory)
	if err != nil {
		return nil, err
	}
	return &OAuthTokenRepository{repo: repo}, nil
}

func (r *OAuthTokenRepository) Save(token *models.OAuthToken, now time.Time) error {
	if token.ID == "" {
		_, err := r.repo.Create(token, now)
		return err
	}
	_, err := r.repo.Update(token, now)
	return err
}

func (r *OAuthTokenRepository) FindByTokenHashAndClientID(tokenHash string, clientID string, now time.Time) (*models.OAuthToken, error) {
	result, err := r.repo.Find("TokenHash="+tokenHash+" & ClientID="+clientID, 1, "", now)
	if err != nil {
		return nil, err
	}
	if result != nil && len(result.Entities) > 0 {
		token := result.Entities[0]
		return &token, nil
	}
	return nil, nil
}

func (r *OAuthTokenRepository) FindByTokenHash(tokenHash string, now time.Time) (*models.OAuthToken, error) {
	result, err := r.repo.Find("TokenHash="+tokenHash, 1, "", now)
	if err != nil {
		return nil, err
	}
	if result != nil && len(result.Entities) > 0 {
		token := result.Entities[0]
		return &token, nil
	}
	return nil, nil
}

func (r *OAuthTokenRepository) Delete(id string, now time.Time) error {
	_, err := r.repo.Delete(id, now)
	return err
}
