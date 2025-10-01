package domain

import "errors"

// UserRecord represents the persistence layer projection of a user.
// It intentionally mirrors the `User` aggregate but stays as a separate
// struct to express storage-level concerns (e.g. soft delete flag).
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
	Delete(userID string) error // 物理削除
}

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)
