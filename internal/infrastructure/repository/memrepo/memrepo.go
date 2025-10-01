package memrepo

import (
	"sync"

	"accountapi/internal/domain"
)

// MemoryRepo stores user records in process memory for testing or lightweight usage.
type MemoryRepo struct {
	mu    sync.RWMutex
	users map[string]*domain.UserRecord
}

// New returns an initialized in-memory repository.
func New() *MemoryRepo {
	return &MemoryRepo{users: make(map[string]*domain.UserRecord)}
}

func clone(rec *domain.UserRecord) *domain.UserRecord {
	if rec == nil {
		return nil
	}
	c := *rec
	return &c
}

func (r *MemoryRepo) Create(rec *domain.UserRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.users[rec.UserID]; exists {
		return domain.ErrAlreadyExists
	}
	r.users[rec.UserID] = clone(rec)
	return nil
}

func (r *MemoryRepo) FindByID(userID string) (*domain.UserRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.users[userID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return clone(rec), nil
}

func (r *MemoryRepo) UpdateProfile(userID, nickname, comment string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.users[userID]
	if !ok {
		return domain.ErrNotFound
	}
	rec.Nickname = nickname
	rec.Comment = comment
	return nil
}

func (r *MemoryRepo) Delete(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[userID]; !ok {
		return domain.ErrNotFound
	}
	delete(r.users, userID)
	return nil
}
