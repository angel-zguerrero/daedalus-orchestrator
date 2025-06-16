package models

// User represents a user account in the system.
// It typically stores information retrieved from the database.
type User struct {
	ID string `orm:"primary-key"`
	// Username is the unique identifier for the user.
	Username string `orm:"unique"`
	// Email is the user's email address.
	Email string `orm:"unique"`
	// PasswordHash is the hashed version of the user's password.
	// Storing plain text passwords is a security risk, so only the hash should be stored.
	PasswordHash string

	IsRootUser bool
}

func (User) TableName() string {
	return "users"
}

// CreateUser is a struct used for capturing user input when creating a new user.
// It includes the plain text password, which should be hashed before being stored
// in the `User` struct's `PasswordHash` field.
type CreateUser struct {
	// Username is the desired username for the new user.
	Username string
	// Email is the email address for the new user.
	Email string
	// Password is the plain text password provided by the user during account creation.
	// This should be processed (hashed) and not stored directly.
	Password string

	IsRootUser bool
}
