package database

import "time"

type User struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type UserRepositoryInterface interface {
	CreateUser(user User) error
	GetUser(id string) (User, error)
	GetUserByUsername(username string) (User, error)
	UpdateUser(user User) error
	DeleteUser(id string) error
}
