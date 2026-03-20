package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/svdx9/go-podcaster/internal/db/queries"
	"github.com/svdx9/go-podcaster/internal/episode/repository"
)

type DBTX = queries.DBTX

type EpisodeRepository struct {
	q queries.Querier
}

func NewEpisodeRepository(querier queries.Querier) *EpisodeRepository {
	return &EpisodeRepository{
		q: querier,
	}
}

func (r *EpisodeRepository) Insert(ctx context.Context, episode repository.Episode) error {
	params := queries.InsertEpisodeParams{
		Uuid:         episode.UUID,
		Title:        episode.Title,
		Description:  episode.Description,
		PubDate:      episode.PubDate,
		Author:       sql.NullString{String: episode.Author, Valid: episode.Author != ""},
		FilePath:     episode.FilePath,
		FileName:     episode.FileName,
		FileSize:     episode.FileSize,
		MimeType:     episode.MimeType,
		DurationSecs: sql.NullInt64{Int64: int64(episode.DurationSecs), Valid: episode.DurationSecs > 0},
	}
	_, err := r.q.InsertEpisode(ctx, params)
	return err
}

func (r *EpisodeRepository) GetByUUID(ctx context.Context, UUID uuid.UUID) (repository.Episode, error) {
	ep, err := r.q.GetEpisodeByUUID(ctx, UUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.Episode{}, repository.ErrNotFound
		}
		return repository.Episode{}, err
	}
	return toDomainEpisode(ep)
}

func (r *EpisodeRepository) List(ctx context.Context, limit, offset int) ([]repository.Episode, error) {
	episodes, err := r.q.ListEpisodes(ctx, queries.ListEpisodesParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, err
	}
	return toDomainEpisodes(episodes)
}

func (r *EpisodeRepository) Delete(ctx context.Context, UUID uuid.UUID) error {
	err := r.q.DeleteEpisode(ctx, UUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *EpisodeRepository) ListAll(ctx context.Context) ([]repository.Episode, error) {
	episodes, err := r.q.ListAllEpisodes(ctx)
	if err != nil {
		return nil, err
	}
	return toDomainEpisodes(episodes)
}

func toDomainEpisode(ep queries.Episode) (repository.Episode, error) {
	d := repository.Episode{
		UUID:         ep.Uuid,
		Title:        ep.Title,
		Description:  ep.Description,
		Author:       ep.Author.String,
		PubDate:      ep.PubDate,
		FilePath:     ep.FilePath,
		FileName:     ep.FileName,
		FileSize:     ep.FileSize,
		MimeType:     ep.MimeType,
		DurationSecs: int(ep.DurationSecs.Int64),
		CreatedAt:    ep.CreatedAt,
	}
	return d, nil
}

func toDomainEpisodes(episodes []queries.Episode) ([]repository.Episode, error) {
	result := make([]repository.Episode, len(episodes))
	for i, ep := range episodes {
		e, err := toDomainEpisode(ep)
		if err != nil {
			return nil, err
		}
		result[i] = e
	}
	return result, nil
}
