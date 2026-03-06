package file

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"faas_goida/internal/auth"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	req, err := parseMultipartCreateRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	projectID, ok := parseProjectIDFromPath(w, r)
	if !ok {
		_ = req.File.Close()
		return
	}
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		_ = req.File.Close()
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	defer func(File multipartFile) {
		err := File.Close()
		if err != nil {
			return
		}
	}(req.File)

	created, err := h.service.Create(r.Context(), userID, CreateRequest{
		Name:        req.Name,
		ProjectID:   projectID,
		UploadName:  req.UploadName,
		ContentType: req.ContentType,
		Size:        req.Size,
		Body:        req.File,
	})
	if err != nil {
		if errors.Is(err, ErrProjectNotFoundFK) {
			writeError(w, http.StatusBadRequest, "project not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) ListByProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	projectID, ok := parseProjectIDFromPath(w, r)
	if !ok {
		return
	}
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := h.service.ListByProject(r.Context(), projectID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, ok := parsePathID(w, r)
	if !ok {
		return
	}
	projectID, ok := parseProjectIDFromPath(w, r)
	if !ok {
		return
	}
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	item, err := h.service.GetByID(r.Context(), id, projectID, userID)
	if err != nil {
		if errors.Is(err, ErrFileNotFound) {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, ok := parsePathID(w, r)
	if !ok {
		return
	}
	projectID, ok := parseProjectIDFromPath(w, r)
	if !ok {
		return
	}
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	req, err := parseMultipartUpdateRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.File != nil {
		defer func(File multipartFile) {
			err := File.Close()
			if err != nil {
				return
			}
		}(req.File)
	}

	updated, err := h.service.Update(r.Context(), userID, UpdateRequest{
		ID:          id,
		ProjectID:   projectID,
		Name:        req.Name,
		UploadName:  req.UploadName,
		ContentType: req.ContentType,
		Size:        req.Size,
		Body:        req.File,
	})
	if err != nil {
		if errors.Is(err, ErrFileNotFound) {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		if errors.Is(err, ErrProjectNotFoundFK) {
			writeError(w, http.StatusBadRequest, "project not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, ok := parsePathID(w, r)
	if !ok {
		return
	}
	projectID, ok := parseProjectIDFromPath(w, r)
	if !ok {
		return
	}
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	err := h.service.Delete(r.Context(), id, projectID, userID)
	if err != nil {
		if errors.Is(err, ErrFileNotFound) {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parsePathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := r.PathValue("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type multipartRequest struct {
	Name        string
	UploadName  string
	ContentType string
	Size        int64
	File        multipartFile
}

type multipartFile interface {
	Read(p []byte) (n int, err error)
	Close() error
}

func parseMultipartCreateRequest(r *http.Request) (multipartRequest, error) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		return multipartRequest{}, ErrInvalidMultipart
	}

	name := strings.TrimSpace(r.FormValue("name"))

	filePart, fileHeader, err := r.FormFile("file")
	if err != nil {
		return multipartRequest{}, errors.New("file is required")
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	uploadName := strings.TrimSpace(fileHeader.Filename)
	if uploadName == "" {
		return multipartRequest{}, errors.New("invalid file name")
	}

	if name == "" {
		name = uploadName
	}

	return multipartRequest{
		Name:        name,
		UploadName:  uploadName,
		ContentType: contentType,
		Size:        fileHeader.Size,
		File:        filePart,
	}, nil
}

func parseMultipartUpdateRequest(r *http.Request) (multipartRequest, error) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		return multipartRequest{}, ErrInvalidMultipart
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		return multipartRequest{}, errors.New("name is required")
	}

	req := multipartRequest{
		Name: name,
	}

	filePart, fileHeader, err := r.FormFile("file")
	if err == nil {
		contentType := fileHeader.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		uploadName := strings.TrimSpace(fileHeader.Filename)
		if uploadName == "" {
			_ = filePart.Close()
			return multipartRequest{}, errors.New("invalid file name")
		}
		req.UploadName = uploadName
		req.ContentType = contentType
		req.Size = fileHeader.Size
		req.File = filePart
	}

	return req, nil
}

func parseProjectIDFromPath(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := r.PathValue("project_id")
	projectID, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || projectID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid project_id")
		return 0, false
	}
	return projectID, true
}
