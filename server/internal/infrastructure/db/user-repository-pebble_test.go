package db_test

import (
	"sync"
	"testing"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// newPebbleStoreForUserRepoTest sets up a PebbleDB instance for user repository testing.
func newPebbleStoreForUserRepoTest(t *testing.T) db.KVStore {
	tmpDir := t.TempDir() // Creates a temporary directory that is automatically cleaned up

	// As per pebble-store.go, CreatePebbleStore handles DB creation and CF setup.
	// UserRepository uses AdminFC. We need to ensure this CF is declared.
	// DefaultFC and MetaFC are already handled by CreatePebbleStore.
	kvStore, err := db.CreatePebbleStore(tmpDir, []string{db.AdminFC}, []string{})
	require.NoError(t, err, "Failed to create PebbleStore")

	t.Cleanup(func() {
		err := kvStore.Close()
		if err != nil {
			// Allow pebble.ErrClosed to be ignored as it means already closed.
			if err.Error() != "pebble: database closed" {
				t.Logf("Warning: error closing pebble store: %v", err)
			}
		}
		// tmpDir is cleaned by t.TempDir() automatically
	})
	return kvStore
}

type TestIDGeneratorFactoryRepositoryPebble struct {
	ids   []string
	index int
	mu    sync.Mutex
}

func (g *TestIDGeneratorFactoryRepositoryPebble) GenerateID() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.ids) == 0 {
		return ""
	}

	id := g.ids[g.index]
	g.index = (g.index + 1) % len(g.ids) // avance circular
	return id
}

func NewTestIDGeneratorFactoryRepositoryPebble(ids []string) *TestIDGeneratorFactoryRepositoryPebble {
	return &TestIDGeneratorFactoryRepositoryPebble{
		ids: ids,
	}
}

// newUserRepoPebbleTest creates a new UnitOfWork and UserRepository for testing with Pebble.
func newUserRepoPebbleTest(t *testing.T) (*db.UnitOfWork, db.KVStore, *db.UserRepository) {
	store := newPebbleStoreForUserRepoTest(t)
	uow := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactoryRepositoryPebble([]string{"123"})
	userRepo, err := db.NewUserRepository(uow, iGF)
	require.NoError(t, err, "Failed to create UserRepository with Pebble backend")
	return uow, store, userRepo
}

// TestPutUser_Success_Pebble tests creating a user successfully.
func TestPebblePutUser_Success_Pebble(t *testing.T) {
	uow, store, repo := newUserRepoPebbleTest(t)

	userToCreate := models.CreateUser{
		Username: "pebbleuser",
		Email:    "pebble@example.com",
		Password: "password123",
	}

	id, err := repo.CreateUser(userToCreate)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	err = uow.Commit()
	require.NoError(t, err)
	iGF := NewTestIDGeneratorFactoryRepositoryPebble([]string{"123"})
	// Verify user is created by reading it back from a new UoW/Repo on the same store
	verifyUOW := db.NewUnitOfWork(store, nil) // Use same store
	verifyRepo, err := db.NewUserRepository(verifyUOW, iGF)
	require.NoError(t, err)

	retrievedUser, err := verifyRepo.GetUserByUsername("pebbleuser")
	require.NoError(t, err)
	require.NotNil(t, retrievedUser)
	assert.Equal(t, userToCreate.Username, retrievedUser.Username)
	assert.Equal(t, userToCreate.Email, retrievedUser.Email)
	assert.NotEmpty(t, retrievedUser.ID)
	assert.Equal(t, id, retrievedUser.ID)

	err = bcrypt.CompareHashAndPassword([]byte(retrievedUser.PasswordHash), []byte(userToCreate.Password))
	assert.NoError(t, err, "Stored password hash should match original password")
}

// TestGetUser_Success_Pebble tests retrieving an existing user.
func TestPebbleGetUser_Success_Pebble(t *testing.T) {
	store := newPebbleStoreForUserRepoTest(t) // Single store for the test
	userToCreate := models.CreateUser{Username: "getme_pebble", Email: "getme@pebble.com", Password: "password"}

	// Create user with initial UoW
	createUOW := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactoryRepositoryPebble([]string{"123"})
	createRepo, err := db.NewUserRepository(createUOW, iGF)
	require.NoError(t, err)
	createdID, err := createRepo.CreateUser(userToCreate)
	require.NoError(t, err)
	require.NotEmpty(t, createdID)
	err = createUOW.Commit()
	require.NoError(t, err)

	// Read user with a new UoW on the same store
	readUOW := db.NewUnitOfWork(store, nil)
	readRepo, err := db.NewUserRepository(readUOW, iGF)
	require.NoError(t, err)

	retrievedUser, err := readRepo.GetUserByUsername(userToCreate.Username)
	require.NoError(t, err)
	require.NotNil(t, retrievedUser)
	assert.Equal(t, userToCreate.Username, retrievedUser.Username)
	assert.Equal(t, userToCreate.Email, retrievedUser.Email)

}

// TestGetUser_NotFound_Pebble tests retrieving a non-existent user.
func TestPebbleGetUser_NotFound_Pebble(t *testing.T) {
	_, _, repo := newUserRepoPebbleTest(t) // Fresh DB, no users

	user, err := repo.GetUserByUsername("nonexistent_pebble_user")
	require.NoError(t, err)
	require.Nil(t, user)

}

// TestDeleteUser_Success_Pebble tests deleting an existing user.
func TestPebbleDeleteUser_Success_Pebble(t *testing.T) {
	store := newPebbleStoreForUserRepoTest(t)
	userToDelete := models.CreateUser{Username: "deleteme_pebble", Email: "deleteme@pebble.com", Password: "password"}

	// Create user
	createUOW := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactoryRepositoryPebble([]string{"123"})
	createRepo, err := db.NewUserRepository(createUOW, iGF)
	require.NoError(t, err)
	_, err = createRepo.CreateUser(userToDelete)
	require.NoError(t, err)
	err = createUOW.Commit()
	require.NoError(t, err)

	// Delete user
	deleteUOW := db.NewUnitOfWork(store, nil)
	deleteRepo, err := db.NewUserRepository(deleteUOW, iGF)
	require.NoError(t, err)
	deleted, err := deleteRepo.DeleteUser(userToDelete.Username)
	require.NoError(t, err)
	require.True(t, deleted)
	err = deleteUOW.Commit()
	require.NoError(t, err)

	// Verify user is deleted
	verifyUOW := db.NewUnitOfWork(store, nil)
	verifyRepo, err := db.NewUserRepository(verifyUOW, iGF)
	require.NoError(t, err)
	goneUser, err := verifyRepo.GetUserByUsername(userToDelete.Username)
	require.NoError(t, err)
	require.Nil(t, goneUser, "User should be deleted")
}

// TestDeleteUser_CannotDeleteRoot_Pebble tests that a root user cannot be deleted.
func TestPebbleDeleteUser_CannotDeleteRoot_Pebble(t *testing.T) {
	store := newPebbleStoreForUserRepoTest(t)
	rootUserToCreate := models.CreateUser{
		Username:   "root_pebble_admin",
		Email:      "root_pebble@example.com",
		Password:   "password",
		IsRootUser: true,
	}

	// Create root user
	createUOW := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactoryRepositoryPebble([]string{"123"})
	createRepo, err := db.NewUserRepository(createUOW, iGF)
	require.NoError(t, err)
	_, err = createRepo.CreateUser(rootUserToCreate)
	require.NoError(t, err)
	err = createUOW.Commit()
	require.NoError(t, err)

	// Attempt to delete
	deleteUOW := db.NewUnitOfWork(store, nil)
	deleteRepo, err := db.NewUserRepository(deleteUOW, iGF)
	require.NoError(t, err)
	deleted, err := deleteRepo.DeleteUser(rootUserToCreate.Username)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete root user")
	assert.False(t, deleted)
	// No commit, as operation should fail

	// Verify user still exists
	verifyUOW := db.NewUnitOfWork(store, nil)
	verifyRepo, err := db.NewUserRepository(verifyUOW, iGF)
	require.NoError(t, err)
	stillRoot, err := verifyRepo.GetUserByUsername(rootUserToCreate.Username)
	require.NoError(t, err)
	require.NotNil(t, stillRoot, "Root user should still exist")
}

// TestDeleteUser_NotFound_Pebble (adapted from GetError)
func TestPebbleDeleteUser_NotFound_Pebble(t *testing.T) {
	_, _, repo := newUserRepoPebbleTest(t) // Fresh DB
	deleted, err := repo.DeleteUser("nonexistent_pebble_user_to_delete")
	require.NoError(t, err) // DeleteUser returns (false, nil) if user not found
	assert.False(t, deleted)
}

// TestLoginUser_Pebble tests various login scenarios.
func TestPebbleLoginUser_Pebble(t *testing.T) {
	store := newPebbleStoreForUserRepoTest(t) // Shared store for all sub-tests

	userEmail := "login_pebble@example.com"
	userUsername := "login_pebble_user"
	correctPassword := "password123"
	incorrectPassword := "wrongpassword"

	// Create the user for login tests
	initialUOW := db.NewUnitOfWork(store, nil)
	iGF := NewTestIDGeneratorFactoryRepositoryPebble([]string{"123"})
	initialRepo, err := db.NewUserRepository(initialUOW, iGF)
	require.NoError(t, err)
	_, err = initialRepo.CreateUser(models.CreateUser{
		Username: userUsername,
		Email:    userEmail,
		Password: correctPassword,
	})
	require.NoError(t, err)
	err = initialUOW.Commit()
	require.NoError(t, err)

	t.Run("LoginWithEmail_CorrectPassword_Success_Pebble", func(t *testing.T) {
		loginUOW := db.NewUnitOfWork(store, nil)
		iGF := NewTestIDGeneratorFactoryRepositoryPebble([]string{"123"})
		loginRepo, err := db.NewUserRepository(loginUOW, iGF)
		require.NoError(t, err)
		loggedIn, err := loginRepo.Login(userEmail, correctPassword)
		require.NoError(t, err)
		assert.True(t, loggedIn)
	})

	t.Run("LoginWithUsername_CorrectPassword_Success_Pebble", func(t *testing.T) {
		loginUOW := db.NewUnitOfWork(store, nil)
		iGF := NewTestIDGeneratorFactoryRepositoryPebble([]string{"123"})
		loginRepo, err := db.NewUserRepository(loginUOW, iGF)
		require.NoError(t, err)
		loggedIn, err := loginRepo.Login(userUsername, correctPassword)
		require.NoError(t, err)
		assert.True(t, loggedIn)
	})

	t.Run("LoginWithEmail_IncorrectPassword_Failure_Pebble", func(t *testing.T) {
		loginUOW := db.NewUnitOfWork(store, nil)
		iGF := NewTestIDGeneratorFactoryRepositoryPebble([]string{"123"})
		loginRepo, err := db.NewUserRepository(loginUOW, iGF)
		require.NoError(t, err)
		loggedIn, err := loginRepo.Login(userEmail, incorrectPassword)
		require.NoError(t, err) // bcrypt mismatch is not an operational error
		assert.False(t, loggedIn)
	})

	t.Run("LoginWithUsername_IncorrectPassword_Failure_Pebble", func(t *testing.T) {
		loginUOW := db.NewUnitOfWork(store, nil)
		iGF := NewTestIDGeneratorFactoryRepositoryPebble([]string{"123"})
		loginRepo, err := db.NewUserRepository(loginUOW, iGF)
		require.NoError(t, err)
		loggedIn, err := loginRepo.Login(userUsername, incorrectPassword)
		require.NoError(t, err) // bcrypt mismatch is not an operational error
		assert.False(t, loggedIn)
	})

	t.Run("Login_UserNotFound_Failure_Pebble", func(t *testing.T) {
		loginUOW := db.NewUnitOfWork(store, nil)
		iGF := NewTestIDGeneratorFactoryRepositoryPebble([]string{"123"})
		loginRepo, err := db.NewUserRepository(loginUOW, iGF)
		require.NoError(t, err)
		unknownIdentifier := "unknown_pebble@example.com"
		loggedIn, err := loginRepo.Login(unknownIdentifier, "anypassword")
		require.NoError(t, err)
		assert.False(t, loggedIn)
	})
}
