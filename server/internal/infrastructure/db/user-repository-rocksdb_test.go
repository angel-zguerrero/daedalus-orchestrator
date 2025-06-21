package db_test

import (
	"testing"

	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/shared/models"

	"github.com/linxGnu/grocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// newRocksdbStoreForUserRepoTest sets up a RocksDB instance for user repository testing.
// It creates a temporary directory for DB data and ensures cleanup.
// It also ensures the necessary column families like db.AdminFC are created.
func newRocksdbStoreForUserRepoTest(t *testing.T) *db.RocksdbStore {
	tmpDir := t.TempDir()
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)
	opts.SetCreateIfMissingColumnFamilies(true)
	goOp := grocksdb.NewDefaultOptions()

	// Define all column families that might be used by UserRepository.
	// db.AdminFC is the primary one based on user-repository_test.go.
	// It's important to list all CFs that will be accessed.
	// Assuming UserRepository primarily uses AdminFC. If it uses others, they should be added here.
	cfNames := []string{db.DefaultFC, db.AdminFC} // db.AdminFC is crucial for UserRepository
	cfOpts := make([]*grocksdb.Options, len(cfNames))
	for i := range cfNames {
		cfOpts[i] = goOp
	}

	rocks, cfHs, err := grocksdb.OpenDbColumnFamilies(opts, tmpDir, cfNames, cfOpts)
	require.NoError(t, err)
	t.Cleanup(func() {
		for _, h := range cfHs {
			h.Destroy()
		}
		rocks.Close()
		opts.Destroy()
		goOp.Destroy()
	})

	cfMap := make(map[string]*grocksdb.ColumnFamilyHandle, len(cfNames))
	for i, name := range cfNames {
		cfMap[name] = cfHs[i]
	}

	// UserRepository does not seem to use TTL column families based on the provided tests.
	// If it did, they would be initialized here.
	ttlCFMap := make(map[string]*grocksdb.ColumnFamilyHandle)

	return &db.RocksdbStore{
		DB:                     rocks,
		ColumnFamilyHandles:    cfMap,
		TTLColumnFamilyHandles: ttlCFMap, // Empty if no TTL CFs are needed for users
	}
}

// newUserRepoTest creates a new UnitOfWork and UserRepository for testing.
func newUserRepoTest(t *testing.T) (*db.UnitOfWork, db.KVStore, *db.UserRepository) {
	store := newRocksdbStoreForUserRepoTest(t)
	uow := db.NewUnitOfWork(store)
	userRepo, err := db.NewUserRepository(uow)
	require.NoError(t, err, "Failed to create UserRepository")
	return uow, store, userRepo
}

func TestRocksDBPutUser_Success(t *testing.T) {
	uow, store, repo := newUserRepoTest(t) // For this test, a single UoW/DB instance is fine

	userToCreate := models.CreateUser{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	// Ensure no root user exists yet to simplify the test, or handle it if one is auto-created
	// (Based on user-repository_test.go, it checks for existing root user)
	// For a real DB test, this check might need actual DB interaction if NewUserRepository creates one.
	// However, the mock test for PutUser checks for *any* root user, not just *the* root user.
	// The actual implementation of IsRootUser seems to be based on a field in the user model.

	id, err := repo.CreateUser(userToCreate)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	err = uow.Commit(time.Now())
	require.NoError(t, err)

	verifyUOW := db.NewUnitOfWork(store) // Use same store
	verifyRepo, err := db.NewUserRepository(verifyUOW)
	require.NoError(t, err)
	retrievedUser, err := verifyRepo.GetUserByUsername("testuser")
	require.NoError(t, err)
	require.NotNil(t, retrievedUser)
	assert.Equal(t, userToCreate.Username, retrievedUser.Username)
	assert.Equal(t, userToCreate.Email, retrievedUser.Email)
	assert.NotEmpty(t, retrievedUser.ID)

	// Verify password hash is stored and not the plain password
	assert.NotEmpty(t, retrievedUser.PasswordHash)
	assert.NotEqual(t, userToCreate.Password, retrievedUser.PasswordHash)
	err = bcrypt.CompareHashAndPassword([]byte(retrievedUser.PasswordHash), []byte(userToCreate.Password))
	assert.NoError(t, err, "Stored password hash should match original password")
}

func TestRocksDBGetUser_Success(t *testing.T) {
	store := newRocksdbStoreForUserRepoTest(t) // Create store once

	userToCreate := models.CreateUser{Username: "getme", Email: "getme@example.com", Password: "password"}

	// Create user with initial UoW
	createUOW := db.NewUnitOfWork(store)
	createRepo, err := db.NewUserRepository(createUOW)
	require.NoError(t, err)
	createdID, err := createRepo.CreateUser(userToCreate)
	require.NoError(t, err)
	require.NotEmpty(t, createdID)
	err = createUOW.Commit(time.Now())
	require.NoError(t, err)

	// Read user with a new UoW on the same store
	readUOW := db.NewUnitOfWork(store)
	readRepo, err := db.NewUserRepository(readUOW)
	require.NoError(t, err)

	retrievedUser, err := readRepo.GetUserByUsername(userToCreate.Username)
	require.NoError(t, err)
	require.NotNil(t, retrievedUser)
	assert.Equal(t, userToCreate.Username, retrievedUser.Username)
	assert.Equal(t, userToCreate.Email, retrievedUser.Email)

}

func TestRocksDBGetUser_NotFound(t *testing.T) {
	// For NotFound, a fresh DB from newUserRepoTest is fine, as it starts empty.
	_, _, repo := newUserRepoTest(t)

	user, err := repo.GetUserByUsername("nonexistentuser")
	require.NoError(t, err) // Expect no error from the repo method itself for not found
	require.Nil(t, user)

}

// TestGetUser_ErrorOnGet: This test is tricky with a real DB.
// A "get error" usually means an I/O problem with the DB itself.
// Such errors are hard to simulate reliably without fault injection into the DB driver.
// For RocksDB, errors might occur if the DB is closed, corrupted, or if there's a permissions issue.
// We can't easily simulate these in a standard test.
// The original mock test `TestGetUser_ErrorOnGet` simulates the KVStore's Get method returning an error.
// With a real DB, if Get fails, it's usually a more catastrophic failure.
// We will skip trying to *force* a DB read error. If such an error naturally occurs, other tests might catch it.
// Instead, "not found" is the primary "negative" case for GetUser.

// TestGetUser_UnmarshalError: This is also hard to replicate with a real DB if the repository
// always marshals data correctly. Corruption would have to happen outside the application's control.
// We assume the repository writes valid JSON. If we wanted to test this, we'd have to:
// 1. Insert valid data.
// 2. Manually access the DB outside the repository to corrupt the JSON string for a user.
// 3. Try to read it via the repository.
// This is too complex for a standard unit test. We will assume data is not corrupted.

func TestRocksDBDeleteUser_Success(t *testing.T) {
	store := newRocksdbStoreForUserRepoTest(t) // Create store once
	userToDelete := models.CreateUser{Username: "deleteme", Email: "deleteme@example.com", Password: "password"}

	// Create user
	createUOW := db.NewUnitOfWork(store)
	createRepo, err := db.NewUserRepository(createUOW)
	require.NoError(t, err)
	createdID, err := createRepo.CreateUser(userToDelete)
	require.NoError(t, err)
	require.NotEmpty(t, createdID)
	err = createUOW.Commit(time.Now())
	require.NoError(t, err)

	// Confirm user exists before delete, using a new UoW
	checkUOW1 := db.NewUnitOfWork(store)
	checkRepo1, err := db.NewUserRepository(checkUOW1)
	require.NoError(t, err)
	_, err = checkRepo1.GetUserByUsername(userToDelete.Username)
	require.NoError(t, err) // Should find the user

	// Delete user
	deleteUOW := db.NewUnitOfWork(store)
	deleteRepo, err := db.NewUserRepository(deleteUOW)
	require.NoError(t, err)
	deleted, err := deleteRepo.DeleteUser(userToDelete.Username)
	require.NoError(t, err)
	require.True(t, deleted)
	err = deleteUOW.Commit(time.Now())
	require.NoError(t, err)

	// Verify user is deleted, using another UoW
	checkUOW2 := db.NewUnitOfWork(store)
	checkRepo2, err := db.NewUserRepository(checkUOW2)
	require.NoError(t, err)
	goneUser, err := checkRepo2.GetUserByUsername(userToDelete.Username)
	require.NoError(t, err)
	require.Nil(t, goneUser, "User should be deleted")
}

func TestRocksDBDeleteUser_CannotDeleteRoot(t *testing.T) {
	store := newRocksdbStoreForUserRepoTest(t) // Create store once
	rootUserToCreate := models.CreateUser{
		Username:   "rootadmin",
		Email:      "root@example.com",
		Password:   "password",
		IsRootUser: true,
	}

	// Create the root user
	createUOW := db.NewUnitOfWork(store)
	createRepo, err := db.NewUserRepository(createUOW)
	require.NoError(t, err)
	rootUserID, err := createRepo.CreateUser(rootUserToCreate)
	require.NoError(t, err)
	require.NotEmpty(t, rootUserID)
	err = createUOW.Commit(time.Now())
	require.NoError(t, err)

	// Attempt to delete the root user
	deleteUOW := db.NewUnitOfWork(store)
	deleteRepo, err := db.NewUserRepository(deleteUOW)
	require.NoError(t, err)

	// Get the created root user to ensure IsRootUser is set (using deleteRepo's context)
	createdRootUser, err := deleteRepo.GetUserByUsername(rootUserToCreate.Username)
	require.NoError(t, err)
	require.NotNil(t, createdRootUser)
	require.True(t, createdRootUser.IsRootUser, "Created user should be a root user")

	deleted, err := deleteRepo.DeleteUser(rootUserToCreate.Username)
	require.Error(t, err) // Expect an error here
	assert.Contains(t, err.Error(), "cannot delete root user")
	assert.False(t, deleted)
	// No commit for deleteUOW as the operation should fail and not stage changes.

	// Verify user still exists
	verifyUOW := db.NewUnitOfWork(store)
	verifyRepo, err := db.NewUserRepository(verifyUOW)
	require.NoError(t, err)
	stillRoot, err := verifyRepo.GetUserByUsername(rootUserToCreate.Username)
	require.NoError(t, err)
	require.NotNil(t, stillRoot, "Root user should still exist")
}

// TestDeleteUser_GetError: Similar to TestGetUser_ErrorOnGet, hard to simulate DB-level read errors.
// The primary "get error" scenario in DeleteUser before actual deletion is "user not found".
func TestRocksDBDeleteUser_NotFound(t *testing.T) {
	// For NotFound, a fresh DB from newUserRepoTest is fine.
	_, _, repo := newUserRepoTest(t)

	deleted, err := repo.DeleteUser("nonexistentuser")
	require.NoError(t, err) // DeleteUser might return (false, nil) if user not found
	assert.False(t, deleted)
	// No commit as nothing should have been deleted.
}

// TestDeleteUser_UnmarshalRootError: Similar to TestGetUser_UnmarshalError. Assumes data integrity.

// TestDeleteUser_WriteError: This tests if uow.Commit() fails after a delete operation.
func TestRocksDBDeleteUser_WriteError(t *testing.T) {
	// This requires a way to make the underlying KVStore's Write/Commit fail.
	// This is hard to achieve with a real RocksDB instance without specific fault injection.
	// The mock test `TestDeleteUser_WriteError` mocks `store.Write` to return an error.
	// We can't directly do that here.
	// One way could be to try to write a malformed batch, but the UOW and repository
	// should prevent this. Or perhaps close the DB before commit?
	// For now, this specific scenario (commit failing *after* a successful delete op staging)
	// is hard to test reliably with the real DB setup without more advanced techniques.
	// We will assume UoW commit either fully works or fully fails for all its operations.
	// If a write error happens, the transaction should roll back.
	t.Skip("Skipping TestDeleteUser_WriteError as it's hard to reliably simulate KVStore write failures with a real DB.")
}

// TestPutUser_KVStorePutError: Similar to TestDeleteUser_WriteError. Tests uow.Commit() failure.
func TestRocksDBPutUser_KVStorePutError(t *testing.T) {
	// Similar rationale to TestDeleteUser_WriteError.
	t.Skip("Skipping TestPutUser_KVStorePutError as it's hard to reliably simulate KVStore write failures with a real DB.")
}

// TestLoginUser tests all login scenarios.
func TestRocksDBLoginUser(t *testing.T) {
	uow, _, repo := newUserRepoTest(t)

	userEmail := "login@example.com"
	userUsername := "loginuser"
	correctPassword := "password123"
	incorrectPassword := "wrongpassword"

	// Create the user for login tests
	createdUser := models.CreateUser{
		Username: userUsername,
		Email:    userEmail,
		Password: correctPassword,
	}
	_, err := repo.CreateUser(createdUser)
	require.NoError(t, err)
	err = uow.Commit(time.Now())
	require.NoError(t, err)

	// Each sub-test should use a fresh UoW and Repo on the *same* underlying store
	// to ensure the user created above is available.
	// However, newUserRepoTest creates a *new* temp DB each time.
	// This means we need to create the user within each sub-test's DB context or share the store.

	// Let's adjust: setup the store once, then create UoW/Repo from it for each sub-test.
	store := newRocksdbStoreForUserRepoTest(t) // Create store once for all sub-tests

	// Create user in this store
	initialUOW := db.NewUnitOfWork(store)
	initialRepo, err := db.NewUserRepository(initialUOW)
	require.NoError(t, err)
	_, err = initialRepo.CreateUser(createdUser)
	require.NoError(t, err)
	err = initialUOW.Commit(time.Now())
	require.NoError(t, err)

	t.Run("LoginWithEmail_CorrectPassword_Success", func(t *testing.T) {
		loginUOW := db.NewUnitOfWork(store)
		loginRepo, err := db.NewUserRepository(loginUOW)
		require.NoError(t, err)

		loggedIn, err := loginRepo.Login(userEmail, correctPassword)
		require.NoError(t, err)
		assert.True(t, loggedIn)
	})

	t.Run("LoginWithUsername_CorrectPassword_Success", func(t *testing.T) {
		loginUOW := db.NewUnitOfWork(store)
		loginRepo, err := db.NewUserRepository(loginUOW)
		require.NoError(t, err)

		loggedIn, err := loginRepo.Login(userUsername, correctPassword)
		require.NoError(t, err)
		assert.True(t, loggedIn)
	})

	t.Run("LoginWithEmail_IncorrectPassword_Failure", func(t *testing.T) {
		loginUOW := db.NewUnitOfWork(store)
		loginRepo, err := db.NewUserRepository(loginUOW)
		require.NoError(t, err)

		loggedIn, err := loginRepo.Login(userEmail, incorrectPassword)
		require.NoError(t, err) // bcrypt mismatch is not a operational error
		assert.False(t, loggedIn)
	})

	t.Run("LoginWithUsername_IncorrectPassword_Failure", func(t *testing.T) {
		loginUOW := db.NewUnitOfWork(store)
		loginRepo, err := db.NewUserRepository(loginUOW)
		require.NoError(t, err)

		loggedIn, err := loginRepo.Login(userUsername, incorrectPassword)
		require.NoError(t, err) // bcrypt mismatch is not a operational error
		assert.False(t, loggedIn)
	})

	t.Run("Login_UserNotFound_Failure", func(t *testing.T) {
		loginUOW := db.NewUnitOfWork(store)
		loginRepo, err := db.NewUserRepository(loginUOW)
		require.NoError(t, err)

		unknownIdentifier := "unknown@example.com"
		loggedIn, err := loginRepo.Login(unknownIdentifier, "anypassword")
		require.NoError(t, err)
		assert.False(t, loggedIn)
	})

	// Login_ErrorOnEmailLookup & Login_ErrorOnUsernameLookup are hard to test with real DB
	// as they imply DB connectivity issues, not just data not found.
	// If the DB connection is fine, these lookups will either find data or return nil (no data).
	// An actual error would be something like DB closed, which is not what these specific tests aim for.
	// These were more relevant for mock testing where you could force the Get method to return an arbitrary error.
	t.Run("Login_ErrorOnEmailLookup", func(t *testing.T) {
		t.Skip("Skipping Login_ErrorOnEmailLookup as it's hard to reliably simulate specific KVStore read errors with a real DB beyond 'not found'.")
	})

	t.Run("Login_ErrorOnUsernameLookup", func(t *testing.T) {
		t.Skip("Skipping Login_ErrorOnUsernameLookup as it's hard to reliably simulate specific KVStore read errors with a real DB beyond 'not found'.")
	})
}
