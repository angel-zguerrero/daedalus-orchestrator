package db_test

import (
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOAuthCredentialsGeneration(t *testing.T) {
	clientID, err := db.GenerateClientID()
	assert.NoError(t, err)
	assert.NotEmpty(t, clientID)

	clientSecret, err := db.GenerateClientSecret()
	assert.NoError(t, err)
	assert.NotEmpty(t, clientSecret)
}

func TestCreateOAuthApp_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewOAuthAppRepository(uow, iGF)
	assert.NoError(t, err)

	input := models.CreateOAuthApp{
		ID:            "123",
		ClientID:      "app_test_client_id",
		ClientSecret:  "sec_plain_secret_123",
		Name:          "Test App",
		TenantID:      "tenant1",
		Description:   "Description",
		AllowedScopes: []string{"queues:read", "queues:write"},
		Now:           time.Now(),
	}

	mockStore.On("Exists", db.AdminFC, db.AdminFCSector, "admin_schema:oauth_apps:data:123", mock.Anything).Return(false, nil)
	mockStore.On("Exists", db.AdminFC, db.AdminFCSector, "admin_schema:oauth_apps:idx-u:ClientID:app_test_client_id", mock.Anything).Return(false, nil)
	mockStore.On("Get", db.AdminFC, db.AdminFCSector, "admin_schema:oauth_apps:grp:TenantID:tenant1", mock.Anything).Return([]byte("[]"), nil)
	mockStore.On("Write", mock.Anything, mock.Anything).Return(nil)

	id, err := repo.CreateApp(input)
	assert.NoError(t, err)
	assert.Equal(t, "123", id)

	err = uow.Commit()
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestValidateOAuthSecret(t *testing.T) {
	plainSecret := "sec_my_secret_token_123"
	hash, err := db.HashClientSecret(plainSecret)
	assert.NoError(t, err)

	app := models.OAuthApp{
		ID:               "123",
		ClientID:         "app_test_client_id",
		ClientSecretHash: hash,
	}

	assert.True(t, db.ValidateOAuthSecret(&app, plainSecret), "Correct secret should validate")
	assert.False(t, db.ValidateOAuthSecret(&app, "sec_wrong_secret"), "Wrong secret should fail")
}
