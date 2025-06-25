package db_test

import (
	"encoding/json"
	"errors"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"

	"golang.org/x/crypto/bcrypt"
)

func TestPutUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	assert.NoError(t, err)

	user := models.CreateUser{Username: "foo", Email: "foo@mail.com", Password: "1234"}

	mockStore.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return(nil, "", nil)
	mockStore.On("Exists", db.AdminFC, "admin_schema:users:idx-u:Username:foo", mock.Anything).Return(false, nil).Times(1)
	mockStore.On("Exists", db.AdminFC, "admin_schema:users:idx-u:Email:foo@mail.com", mock.Anything).Return(false, nil).Times(1)
	mockStore.On("Exists", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(false, nil).Times(1)
	mockStore.On("Write", mock.Anything, mock.Anything).Return(nil).Times(1)

	id, err := repo.CreateUser(user)
	assert.NoError(t, err)
	err = uow.Commit()
	assert.NotNil(t, id)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestGetUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)

	u := models.User{Username: "foo", Email: "bar"}
	data, _ := json.Marshal(u)

	//mockStore.On("Exists", db.AdminFC, "admin_schema:users:idx-u:Username:foo", mock.Anything).Return(true, nil)
	//mockStore.On("Exists", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(true, nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:foo", mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(data, nil)

	user, err := repo.GetUserByUsername("foo")
	assert.NoError(t, err)
	assert.Equal(t, "foo", user.Username)
	mockStore.AssertExpectations(t)
}

func TestGetUser_NotFound(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:bar", mock.Anything).Return(nil, nil)

	user, err := repo.GetUserByUsername("bar")
	assert.NoError(t, err)
	assert.Nil(t, user)
	mockStore.AssertExpectations(t)
}

func TestGetUser_ErrorOnGet(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)

	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:x", mock.Anything).Return(nil, errors.New("get failed"))
	user, err := repo.GetUserByUsername("x")
	assert.Error(t, err)
	assert.Nil(t, user)
	mockStore.AssertExpectations(t)
}

func TestGetUser_UnmarshalError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)

	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:x", mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return([]byte("invalid-json"), nil)

	user, err := repo.GetUserByUsername("x")
	assert.Error(t, err)
	assert.Empty(t, user)
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_Success(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	root := models.User{Username: "other", ID: "123"}
	rootData, _ := json.Marshal(root)

	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:bob", mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(rootData, nil)

	result, err := repo.DeleteUser("bob")
	assert.Equal(t, true, result)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_CannotDeleteRoot(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	root := models.User{Username: "admin", ID: "123", IsRootUser: true}
	rootData, _ := json.Marshal(root)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:admin", mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(rootData, nil)
	_, err = repo.DeleteUser("admin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete root user")
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_GetError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	mockStore.On("Get", db.AdminFC, mock.Anything, mock.Anything).Return(nil, errors.New("get failed"))

	_, err = repo.DeleteUser("someone")
	assert.Error(t, err)
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_UnmarshalRootError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:x", mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return([]byte("invalid-json"), nil)
	_, err = repo.DeleteUser("x")
	assert.Error(t, err)
	mockStore.AssertExpectations(t)
}

func TestDeleteUser_WriteError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	root := models.User{Username: "user", ID: "123"}
	rootData, _ := json.Marshal(root)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:user", mock.Anything).Return([]byte("123"), nil)
	mockStore.On("Get", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(rootData, nil)

	mockStore.On("Write", mock.Anything, mock.Anything).Return(errors.New("write failed"))

	_, err = repo.DeleteUser("user")
	assert.NoError(t, err)
	err = uow.Commit()
	assert.Error(t, err)
	mockStore.AssertExpectations(t)
}
func TestPutUser_KVStorePutError(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	userInput := models.CreateUser{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
	}

	mockStore.On("SearchByPatternPaginatedKV", db.AdminFC, "admin_schema:users:idx:IsRootUser:true:*", "", 1, mock.Anything).Return(nil, "", nil)
	mockStore.On("Exists", db.AdminFC, "admin_schema:users:idx-u:Username:testuser", mock.Anything).Return(false, nil).Times(1)
	mockStore.On("Exists", db.AdminFC, "admin_schema:users:idx-u:Email:test@example.com", mock.Anything).Return(false, nil).Times(1)
	mockStore.On("Exists", db.AdminFC, "admin_schema:users:data:123", mock.Anything).Return(false, nil).Times(1)
	mockStore.On("Write", mock.Anything, mock.Anything).Return(errors.New("kv put failed")).Times(1)

	_, err = repo.CreateUser(userInput)

	assert.NoError(t, err)
	err = uow.Commit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "kv put failed")
	mockStore.AssertExpectations(t)
}

func TestLoginUser(t *testing.T) {
	mockStore := new(MockKVStore)
	uow := db.NewUnitOfWork(mockStore, nil)
	iGF := NewTestIDGeneratorFactory([]string{"123"})
	repo, err := db.NewUserRepository(uow, iGF)
	assert.NoError(t, err)

	userEmail := "test@example.com"
	userUsername := "testuser"
	correctPassword := "password123"
	incorrectPassword := "wrongpassword"
	userID := "user123"

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.DefaultCost)

	user := models.User{
		ID:           userID,
		Username:     userUsername,
		Email:        userEmail,
		PasswordHash: string(hashedPassword),
	}
	userData, _ := json.Marshal(user)

	t.Run("LoginWithEmail_CorrectPassword_Success", func(t *testing.T) {
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+userEmail, mock.Anything).Return([]byte(userID), nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:data:"+userID, mock.Anything).Return(userData, nil).Once()

		found, err := repo.Login(userEmail, correctPassword)
		assert.NoError(t, err)
		assert.True(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("LoginWithUsername_CorrectPassword_Success", func(t *testing.T) {
		// Mock email lookup to return nil, then username lookup to return the user
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+userUsername, mock.Anything).Return(nil, nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:"+userUsername, mock.Anything).Return([]byte(userID), nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:data:"+userID, mock.Anything).Return(userData, nil).Once()

		found, err := repo.Login(userUsername, correctPassword)
		assert.NoError(t, err)
		assert.True(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("LoginWithEmail_IncorrectPassword_Failure", func(t *testing.T) {
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+userEmail, mock.Anything).Return([]byte(userID), nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:data:"+userID, mock.Anything).Return(userData, nil).Once()

		found, err := repo.Login(userEmail, incorrectPassword)
		assert.NoError(t, err) // bcrypt.ErrMismatchedHashAndPassword is not an "error" for Login logic, it's a valid outcome
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("LoginWithUsername_IncorrectPassword_Failure", func(t *testing.T) {
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+userUsername, mock.Anything).Return(nil, nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:"+userUsername, mock.Anything).Return([]byte(userID), nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:data:"+userID, mock.Anything).Return(userData, nil).Once()

		found, err := repo.Login(userUsername, incorrectPassword)
		assert.NoError(t, err) // bcrypt.ErrMismatchedHashAndPassword is not an "error" for Login logic
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("Login_UserNotFound_Failure", func(t *testing.T) {
		unknownIdentifier := "unknown@example.com"
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+unknownIdentifier, mock.Anything).Return(nil, nil).Once()
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:"+unknownIdentifier, mock.Anything).Return(nil, nil).Once()

		found, err := repo.Login(unknownIdentifier, "anypassword")
		assert.NoError(t, err)
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("Login_ErrorOnEmailLookup", func(t *testing.T) {
		errorIdentifier := "error@example.com"
		expectedError := errors.New("db error on email lookup")
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+errorIdentifier, mock.Anything).Return(nil, expectedError).Once()

		found, err := repo.Login(errorIdentifier, "anypassword")
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("Login_ErrorOnUsernameLookup", func(t *testing.T) {
		errorIdentifier := "erroruser"
		expectedError := errors.New("db error on username lookup")
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Email:"+errorIdentifier, mock.Anything).Return(nil, nil).Once() // Email lookup fine, returns nil
		mockStore.On("Get", db.AdminFC, "admin_schema:users:idx-u:Username:"+errorIdentifier, mock.Anything).Return(nil, expectedError).Once()

		found, err := repo.Login(errorIdentifier, "anypassword")
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	// Test case for bcrypt error other than ErrMismatchedHashAndPassword (though hard to simulate without specific bcrypt internals)
	// For now, the existing error handling in Login method should cover general errors from bcrypt.CompareHashAndPassword
}
