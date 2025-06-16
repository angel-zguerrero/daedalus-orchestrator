package db

import (
	"fmt"

	models "deadalus-orch/shared/models"

	"golang.org/x/crypto/bcrypt"
)

type UserRepository struct {
	repo *Repository[models.User]
}

func NewUserRepository(uow *UnitOfWork) (*UserRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.User](uow, AdminFC, "admin_schema", &DefaultIDGeneratorFactory{})
	if err != nil {
		return nil, err
	}
	return &UserRepository{repo: repo}, nil
}

func (r *UserRepository) CreateUser(input models.CreateUser) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	rootUser, err := r.repo.FindByField("IsRootUser", "true")
	if err != nil {
		return "", err
	}

	if rootUser != nil {
		return "", fmt.Errorf("Only a single root user is allowed")
	}

	user := &models.User{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hash),
		IsRootUser:   input.IsRootUser,
	}

	return r.repo.Create(user)
}

func (r *UserRepository) GetUserByUsername(username string) (*models.User, error) {
	return r.repo.FindByField("Username", username)
}

func (r *UserRepository) GetUserRoot() (*models.User, error) {
	return r.repo.FindByField("IsRootUser", "true")
}

func (r *UserRepository) DeleteUser(username string) (bool, error) {
	rootUser, err := r.repo.FindByField("Username", username)
	if err != nil || rootUser == nil {
		return false, err
	}
	if err != nil {
		return false, err
	}

	if rootUser != nil && rootUser.IsRootUser {
		return false, fmt.Errorf("cannot delete root user: %s", username)
	}

	return r.repo.Delete(rootUser.ID)
}
