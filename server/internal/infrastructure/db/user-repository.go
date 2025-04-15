package db

import (
	"encoding/json"
	"fmt"

	"github.com/linxGnu/grocksdb"
	"golang.org/x/crypto/bcrypt"

	constants "deadalus-orch/shared/constants"
	models "deadalus-orch/shared/models"
)

func PutUser(kvStore KVStore, input models.CreateUser) error {
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

	key := "user:" + input.Username
	err = kvStore.Put(wo, []byte(key), userData)
	return err
}

func GetUser(kvStore KVStore, username string) (*models.User, error) {
	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	key := []byte("user:" + username)
	value, err := kvStore.Get(ro, key)
	if err != nil {
		return nil, err
	}
	defer value.Free()

	if !value.Exists() {
		return nil, nil
	}

	var user models.User
	err = json.Unmarshal(value.Data(), &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func GetDefaultRootUserRoot(kvStore KVStore) (*models.CreateUser, error) {
	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	key := []byte(constants.DefaultRootUserRootKey)
	value, err := kvStore.Get(ro, key)
	if err != nil {
		return nil, err
	}
	defer value.Free()

	if !value.Exists() {
		return nil, nil
	}

	var user models.CreateUser
	err = json.Unmarshal(value.Data(), &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

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

	if err := kvStore.Write(wo, batch); err != nil {
		return err
	}

	return nil
}

func DeleteUser(kvStore KVStore, username string) error {
	ro := grocksdb.NewDefaultReadOptions()
	defer ro.Destroy()

	wo := grocksdb.NewDefaultWriteOptions()
	defer wo.Destroy()

	rootData, err := kvStore.Get(ro, []byte(constants.DefaultRootUserRootKey))
	if err != nil {
		return err
	}
	defer rootData.Free()

	if rootData.Exists() {
		var rootUser models.User
		if err := json.Unmarshal(rootData.Data(), &rootUser); err != nil {
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

	if err := kvStore.Write(wo, batch); err != nil {
		return err
	}
	return nil
}
