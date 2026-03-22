package repositorytest

import (
	"context"

	"github.com/google/uuid"
	"github.com/svdx9/go-podcaster/internal/episode/repository"
)

// MockRepository is a test double for repository.Repository.
// It is stateful: Insert appends, Delete removes. Pre-populate Episodes
// and set Err to control return values.
type MockRepository struct {
	Episodes []repository.Episode
	Err      error
}

func (m *MockRepository) Insert(_ context.Context, episode repository.Episode) error {
	if m.Err != nil {
		return m.Err
	}
	m.Episodes = append(m.Episodes, episode)
	return nil
}

func (m *MockRepository) GetByUUID(_ context.Context, id uuid.UUID) (repository.Episode, error) {
	for _, ep := range m.Episodes {
		if ep.UUID == id {
			return ep, m.Err
		}
	}
	return repository.Episode{}, repository.ErrNotFound
}

func (m *MockRepository) List(_ context.Context, limit, offset int) ([]repository.Episode, error) {
	if offset >= len(m.Episodes) {
		return []repository.Episode{}, m.Err
	}
	end := offset + limit
	if end > len(m.Episodes) {
		end = len(m.Episodes)
	}
	return m.Episodes[offset:end], m.Err
}

func (m *MockRepository) Delete(_ context.Context, id uuid.UUID) error {
	if m.Err != nil {
		return m.Err
	}
	for i, ep := range m.Episodes {
		if ep.UUID == id {
			m.Episodes = append(m.Episodes[:i], m.Episodes[i+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

func (m *MockRepository) ListAll(_ context.Context) ([]repository.Episode, error) {
	return m.Episodes, m.Err
}

func (m *MockRepository) ListAllValid(_ context.Context) ([]repository.Episode, error) {
	episodes := []repository.Episode{}
	for _, ep := range m.Episodes {
		if ep.DurationSecs > 0 {
			episodes = append(episodes, ep)
		}
	}
	return episodes, m.Err
}

func (m *MockRepository) UpdateDuration(_ context.Context, _ uuid.UUID, _ int) error {
	return m.Err
}

func (m *MockRepository) ListPendingDuration(_ context.Context) ([]repository.Episode, error) {
	return nil, m.Err
}
