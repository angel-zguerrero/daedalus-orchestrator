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
	repo, err := GetRepository[models.User](uow, AdminFC, AdminFCSector, "admin_schema", factory)
	if err != nil {
		return nil, err
	}
	return &UserRepository{repo: repo}, nil
}

func (r *UserRepository) CreateUser(input models.CreateUser, now time.Time) (string, error) {

	if input.IsRootUser {
		rootUser, err := r.repo.FindByField("IsRootUser", "true", now)
		if err != nil {
			return "", err
		}

		if rootUser != nil {
			return "", fmt.Errorf("Only a single root user is allowed")
		}
	}

	user := &models.User{
		ID:           input.ID,
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: input.Password,
		IsRootUser:   input.IsRootUser,
	}

	return r.repo.Create(user, now)
}

func (r *UserRepository) GetUserByUsername(username string, now time.Time) (*models.User, error) {
	return r.repo.FindByField("Username", username, now)
}

func (r *UserRepository) GetUserRoot(now time.Time) (*models.User, error) {
	return r.repo.FindByField("IsRootUser", "true", now)
}

func (r *UserRepository) GetUsers(filter string, cursor string, limit int, now time.Time) (*FindResult[models.User], error) {
	if filter == "" {
		return r.repo.Find("ID != 0", limit, cursor, now) // ID != 0 Workaround
	} else {
		return r.repo.Find("Username LIKE *"+filter+"* | Email LIKE *"+filter+"*", limit, cursor, now)
	}
}

func (r *UserRepository) UpdateUser(input models.UpdateUser, now time.Time) (bool, error) {
	user, err := r.repo.FindByField("ID", input.ID, now)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, fmt.Errorf("user not found: %s", input.ID)
	}

	user.Username = input.Username
	user.Email = input.Email
	fmt.Printf("UserRepository.UpdateUser received input.Password length: %d\n", len(input.Password))
	if input.Password != "" {
		user.PasswordHash = input.Password
		fmt.Printf("UserRepository.UpdateUser assigned new PasswordHash\n")

		// If root user changes their password, resolve demo mode
		if user.IsRootUser {
			r.repo.kvStore.Put(AdminFC, AdminFCSector, "demo_resolved", []byte("true"), 0, now)
			fmt.Printf("UserRepository.UpdateUser resolved demo mode\n")
		}
	} else {
		fmt.Printf("UserRepository.UpdateUser skipped PasswordHash assignment\n")
	}

	return r.repo.Update(user, now)
}

func (r *UserRepository) DeleteUser(username string, now time.Time) (bool, error) {
	rootUser, err := r.repo.FindByField("Username", username, now)
	if err != nil || rootUser == nil {
		return false, err
	}

	if rootUser != nil && rootUser.IsRootUser {
		return false, fmt.Errorf("cannot delete root user: %s", username)
	}

	return r.repo.Delete(rootUser.ID, now)
}

func (r *UserRepository) Login(usernameOrEmail, password string, now time.Time) (bool, error) {
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
