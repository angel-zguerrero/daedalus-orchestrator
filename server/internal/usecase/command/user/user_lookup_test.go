package user_command_test

import (
	"os"
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/db"
	user_command "deadalus-orch/server/internal/usecase/command/user"
	"deadalus-orch/shared/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestUserPebbleStore(t *testing.T) db.KVStore {
	tempDir, err := os.MkdirTemp("", "user_lookup_test_*")
	require.NoError(t, err)

	cfNames := []string{"default", db.AdminFC}
	store, err := db.CreatePebbleStore(tempDir, cfNames, []string{})
	require.NoError(t, err)

	t.Cleanup(func() {
		store.Close()
		os.RemoveAll(tempDir)
	})
	return store
}

func TestGetUserByIdAndIdsCommands(t *testing.T) {
	store := newTestUserPebbleStore(t)
	uow := db.NewUnitOfWork(store, nil)
	now := time.Now()

	createCmd1 := &user_command.CreateUserCommand{
		ID:         "user-uuid-1",
		Username:   "alice",
		Email:      "alice@example.com",
		Password:   "secret123",
		IsRootUser: false,
	}
	res1 := createCmd1.Execute(uow, now)
	require.Empty(t, res1.Error)

	createCmd2 := &user_command.CreateUserCommand{
		ID:         "user-uuid-2",
		Username:   "bob",
		Email:      "bob@example.com",
		Password:   "secret456",
		IsRootUser: false,
	}
	res2 := createCmd2.Execute(uow, now)
	require.Empty(t, res2.Error)
	require.NoError(t, uow.Commit())

	// Test GetUserByIdCommand by ID
	uowRead := db.NewUnitOfWork(store, nil)
	getByIdCmd := &user_command.GetUserByIdCommand{ID: "user-uuid-1"}
	getByIdRes := getByIdCmd.Execute(uowRead, now)
	require.Empty(t, getByIdRes.Error)
	user1, ok := getByIdRes.Result.(models.User)
	require.True(t, ok)
	assert.Equal(t, "user-uuid-1", user1.ID)
	assert.Equal(t, "alice", user1.Username)
	assert.Empty(t, user1.PasswordHash, "PasswordHash must be scrubbed")

	// Test GetUserByIdCommand fallback by Username
	getByUsernameCmd := &user_command.GetUserByIdCommand{ID: "bob"}
	getByUsernameRes := getByUsernameCmd.Execute(uowRead, now)
	require.Empty(t, getByUsernameRes.Error)
	user2, ok := getByUsernameRes.Result.(models.User)
	require.True(t, ok)
	assert.Equal(t, "user-uuid-2", user2.ID)
	assert.Equal(t, "bob", user2.Username)
	assert.Empty(t, user2.PasswordHash, "PasswordHash must be scrubbed")

	// Test GetUsersByIdsCommand
	getByIdsCmd := &user_command.GetUsersByIdsCommand{
		IDs: []string{"user-uuid-1", "bob", "nonexistent", "user-uuid-1"},
	}
	getByIdsRes := getByIdsCmd.Execute(uowRead, now)
	require.Empty(t, getByIdsRes.Error)
	users, ok := getByIdsRes.Result.([]models.User)
	require.True(t, ok)
	assert.Len(t, users, 2)
	for _, u := range users {
		assert.Empty(t, u.PasswordHash, "PasswordHash must be scrubbed for all batch users")
	}
}
