package domain

import "errors"

type UserRecord struct {
	UserID       string
	PasswordHash string
	Nickname     string
	Comment      string
	Deleted      bool
}

type UserRepository interface {
	Create(rec *UserRecord) error
	FindByID(userID string) (*UserRecord, error)
	UpdateProfile(userID, nickname, comment string) error
	Delete(userID string) error
}

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)
