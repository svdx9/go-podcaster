package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	apiv1 "github.com/svdx9/go-podcaster/internal/api/v1"
	"github.com/svdx9/go-podcaster/internal/episode/repository"
	"github.com/svdx9/go-podcaster/internal/episode/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) PostV1Episodes(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "failed to parse multipart form")
		return
	}

	description := r.FormValue("description")
	if description == "" {
		h.writeError(w, http.StatusBadRequest, "missing_field", "description is required")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "missing_file", "file is required")
		return
	}

	if file == nil {
		h.writeError(w, http.StatusBadRequest, "missing_file", "file is required")
		return
	}
	defer func() { _ = file.Close() }()

	title := r.FormValue("title")
	author := r.FormValue("author")
	pubDateStr := r.FormValue("pub_date")

	req := service.UploadRequest{
		Title:       title,
		Description: description,
		Author:      author,
		PubDate:     pubDateStr,
		File:        file,
		Filename:    header.Filename,
		FileSize:    header.Size,
	}

	episode, err := h.svc.Upload(r.Context(), req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(episodeToResponse(episode))
	if err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func (h *Handler) GetV1Episodes(w http.ResponseWriter, r *http.Request, params apiv1.GetV1EpisodesParams) {
	limit := 20
	if params.Limit != nil && *params.Limit > 0 {
		limit = *params.Limit
	}

	offset := 0
	if params.Offset != nil && *params.Offset >= 0 {
		offset = *params.Offset
	}

	episodes, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	response := make([]apiv1.Episode, len(episodes))
	for i, ep := range episodes {
		response[i] = episodeToResponse(ep)
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func (h *Handler) DeleteV1EpisodesUuid(w http.ResponseWriter, r *http.Request, uuid openapi_types.UUID) {
	err := h.svc.Delete(r.Context(), uuid.String())
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		h.writeError(w, http.StatusNotFound, "not_found", "episode not found")
		return
	}
	if errors.Is(err, service.ErrMissingTitle) {
		h.writeError(w, http.StatusBadRequest, "missing_title", "title is required")
		return
	}
	if errors.Is(err, service.ErrUnsupportedMedia) {
		h.writeError(w, 415, "unsupported_media", "file type not supported")
		return
	}
	h.writeError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(apiv1.Error{Code: code, Message: message})
	if err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

func episodeToResponse(ep repository.Episode) apiv1.Episode {
	resp := apiv1.Episode{
		Uuid:         uuid.MustParse(ep.UUID),
		Title:        ep.Title,
		Description:  ep.Description,
		FileName:     ep.FileName,
		FileSize:     int(ep.FileSize),
		MimeType:     ep.MimeType,
		PubDate:      ep.PubDate,
		CreatedAt:    ep.CreatedAt,
		Author:       &ep.Author,
		DurationSecs: &ep.DurationSecs,
	}
	return resp
}
