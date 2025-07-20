package db

import (
	"fmt"
	"time"

	models "deadalus-orch/shared/models"

	"golang.org/x/crypto/bcrypt"
)

type UserRepository struct {
	repo *Repository[models.User]
}

func NewUserRepository(uow *UnitOfWork, factory IDGeneratorFactory) (*UserRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.User](uow, AdminFC, AdminFCSelector, "admin_schema", factory)
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
	now := time.Now()

	rootUser, err := r.repo.FindByField("IsRootUser", "true", now)
	if err != nil {
		return "", err
	}

	if rootUser != nil {
		return "", fmt.Errorf("Only a single root user is allowed")
	}

	user := &models.User{
		ID:           input.ID,
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: string(hash),
		IsRootUser:   input.IsRootUser,
	}

	return r.repo.Create(user, now)
}

func (r *UserRepository) GetUserByUsername(username string) (*models.User, error) {
	return r.repo.FindByField("Username", username, time.Now())
}

func (r *UserRepository) GetUserRoot() (*models.User, error) {
	return r.repo.FindByField("IsRootUser", "true", time.Now())
}

func (r *UserRepository) DeleteUser(username string) (bool, error) {
	now := time.Now()
	rootUser, err := r.repo.FindByField("Username", username, now)
	if err != nil || rootUser == nil {
		return false, err
	}
	if err != nil {
		return false, err
	}

	if rootUser != nil && rootUser.IsRootUser {
		return false, fmt.Errorf("cannot delete root user: %s", username)
	}

	return r.repo.Delete(rootUser.ID, now)
}

func (r *UserRepository) Login(usernameOrEmail, password string) (bool, error) {
	now := time.Now()
	user, err := r.repo.FindByField("Email", usernameOrEmail, now)
	if err != nil {
		// Error during email lookup
		return false, err
	}

	if user == nil {
		// User not found by email, try by username
		user, err = r.repo.FindByField("Username", usernameOrEmail, now)
		if err != nil {
			// Error during username lookup
			return false, err
		}
	}

	if user == nil {
		// User not found by either email or username
		return false, nil
	}

	// User found, now validate password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			// Password does not match
			return false, nil
		}
		// Some other error during password comparison
		return false, err
	}

	// Password matches
	return true, nil
}
