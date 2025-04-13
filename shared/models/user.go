package models

type User struct {
	Username     string
	Email        string
	PasswordHash string
}

type CreateUser struct {
	Username string
	Email    string
	Password string
}
