package db

import (
	"encoding/json"
	"fmt"

	"github.com/linxGnu/grocksdb"
	"golang.org/x/crypto/bcrypt"

	constants "deadalus-orch/shared/constants"
	models "deadalus-orch/shared/models"
)

// PutUser creates or updates a user in the KVStore.
// It hashes the user's password using bcrypt before storing it.
// The user is stored in the AdminFC column family with a key prefix "user:".
//
// Parameters:
//   - kvStore: The KVStore implementation where user data is stored.
//   - input: A models.CreateUser struct containing the user's details (Username, Email, Password).
//
// Returns:
//   - An error if password hashing fails, JSON marshaling fails, or the Put operation on the KVStore fails.
//     Returns nil on successful user creation/update.
func PutUser(kvStore KVStore, input models.CreateUser) error {

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := models.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hash),
	}

	userData, err := json.Marshal(user)
	if err != nil {
		return err
	}

	key := "user:" + input.Username
	err = kvStore.Put(AdminFC, key, userData)
	return err
}

// GetUser retrieves a user by username from the KVStore.
// It looks for the user in the AdminFC column family using the key "user:<username>".
//
// Parameters:
//   - kvStore: The KVStore implementation where user data is stored.
//   - username: The username of the user to retrieve.
//
// Returns:
//   - A pointer to a models.User struct if the user is found.
//   - nil if the user is not found.
//   - An error if the Get operation on the KVStore fails or if JSON unmarshaling fails.
func GetUser(kvStore KVStore, username string) (*models.User, error) {
	key := "user:" + username
	value, err := kvStore.Get(AdminFC, key)
	if err != nil {
		return nil, err
	}

	if value == nil {
		return nil, nil
	}

	var user models.User
	err = json.Unmarshal(value, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetDefaultRootUserRoot retrieves the initial root user creation details from the KVStore.
// This information is typically stored during the bootstrap process and might be used to
// ensure the root user exists or to re-create it if necessary.
// It looks for the data in the AdminFC column family using the key constants.DefaultRootUserRootKey.
//
// Parameters:
//   - kvStore: The KVStore implementation where user data is stored.
//
// Returns:
//   - A pointer to a models.CreateUser struct if the root user creation data is found.
//   - nil if the data is not found.
//   - An error if the Get operation on the KVStore fails or if JSON unmarshaling fails.
func GetDefaultRootUserRoot(kvStore KVStore) (*models.CreateUser, error) {
	key := constants.DefaultRootUserRootKey
	value, err := kvStore.Get(AdminFC, key)
	if err != nil {
		return nil, err
	}

	if value == nil {
		return nil, nil
	}

	var user models.CreateUser
	err = json.Unmarshal(value, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// PutDefaultRootUserRoot stores the details for creating the default root user and also creates/updates
// the root user itself in a single atomic write batch.
// It stores the raw `models.CreateUser` input under `constants.DefaultRootUserRootKey` and
// the processed `models.User` (with hashed password) under "user:<username>".
// Both operations are performed within the AdminFC column family.
//
// Parameters:
//   - kvStore: The KVStore implementation where user data is stored.
//   - input: A models.CreateUser struct containing the root user's details.
//
// Returns:
//   - An error if password hashing fails, JSON marshaling fails, or the batch Write operation on the KVStore fails.
//     Returns nil on success.
func PutDefaultRootUserRoot(kvStore KVStore, input models.CreateUser) error {
	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := models.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hash),
	}

	userData, err := json.Marshal(user)
	if err != nil {
		return err
	}

	defaultUserData, err := json.Marshal(input)
	if err != nil {
		return err
	}

	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()
	batch.Put([]byte(constants.DefaultRootUserRootKey), defaultUserData)
	userKey := fmt.Sprintf("user:%s", input.Username)
	batch.Put([]byte(userKey), userData)

	if err := kvStore.Write(batch); err != nil {
		return err
	}

	return nil
}

// DeleteUser removes a user from the KVStore.
// It prevents the deletion of the default root user.
// The user is deleted from the AdminFC column family using a key prefix "user:".
// The deletion is performed in a batch, though currently it only contains one delete operation.
//
// Parameters:
//   - kvStore: The KVStore implementation where user data is stored.
//   - username: The username of the user to delete.
//
// Returns:
//   - An error if trying to delete the root user, if the Get operation for root user check fails,
//     if JSON unmarshaling of root user data fails, or if the batch Write operation on the KVStore fails.
//     Returns nil on successful user deletion.
func DeleteUser(kvStore KVStore, username string) error {
	rootData, err := kvStore.Get(AdminFC, constants.DefaultRootUserRootKey)
	if err != nil {
		return err
	}

	if rootData != nil {
		var rootUser models.User
		if err := json.Unmarshal(rootData, &rootUser); err != nil {
			return err
		}
		if rootUser.Username == username {
			return fmt.Errorf("❌ cannot delete root user (%s)", username)
		}
	}

	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()

	userKey := fmt.Sprintf("user:%s", username)
	batch.Delete([]byte(userKey))

	if err := kvStore.Write(batch); err != nil {
		return err
	}
	return nil
}
