package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"faas_goida/internal/auth"
	"faas_goida/internal/file"
)

type Storage interface {
	Download(ctx context.Context, key string) ([]byte, error)
}

type Handler struct {
	files   file.Repository
	storage Storage
	cache   *Cache
}

type callRequest struct {
	ProjectID int64 `json:"project_id"`
	FileID    int64 `json:"file_id"`
}

func NewHandler(files file.Repository, storage Storage, cache *Cache) *Handler {
	if cache == nil {
		cache = NewCache()
	}
	return &Handler{
		files:   files,
		storage: storage,
		cache:   cache,
	}
}

func (h *Handler) Call(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req callRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ProjectID <= 0 || req.FileID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid project_id or file_id")
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	f, err := h.files.GetByID(r.Context(), req.FileID, req.ProjectID, userID)
	if err != nil {
		if errors.Is(err, file.ErrFileNotFound) {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if strings.TrimSpace(f.S3Key) == "" {
		writeError(w, http.StatusBadRequest, "file content is not available")
		return
	}

	content, ok := h.cache.Get(f.S3Key)
	if !ok {
		content, err = h.storage.Download(r.Context(), f.S3Key)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		h.cache.Set(f.S3Key, content, 24*time.Hour)
	}

	filePath, cleanup, err := writeTempFile(f.Name, content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer cleanup()

	cmd := exec.Command("./bin/goida_lang", "run", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error running goida_lang: %v", err)
		log.Printf("goida_lang output: %s", string(output))

		w.WriteHeader(http.StatusInternalServerError)
		if _, err = fmt.Fprintf(w, "Server error (Exit Status: %v)\nDetails: %s", err, string(output)); err != nil {
			log.Fatal(err)
		}
		return
	}

	log.Println("goida_lang finished successfully")
	if _, err := fmt.Fprintf(w, "Result:\n%s", string(output)); err != nil {
		log.Fatal(err)
	}
}

func writeTempFile(name string, data []byte) (string, func(), error) {
	dir, err := os.MkdirTemp("", "goida-run-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(dir)
	}

	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == string(os.PathSeparator) {
		base = "main.goida"
	}
	if filepath.Ext(base) == "" {
		base += ".goida"
	}
	path := filepath.Join(dir, base)
	if err := os.WriteFile(path, data, 0644); err != nil {
		cleanup()
		return "", nil, err
	}
	return path, cleanup, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
