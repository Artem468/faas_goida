package file

import (
	"context"
	"errors"
	"io"
)

type Storage interface {
	Upload(ctx context.Context, projectID int64, fileName string, body io.Reader, size int64, contentType string) (key string, url string, err error)
	Delete(ctx context.Context, key string) error
}

type Service interface {
	Create(ctx context.Context, userID int64, req CreateRequest) (File, error)
	ListByProject(ctx context.Context, projectID, userID int64) ([]File, error)
	GetByID(ctx context.Context, id, projectID, userID int64) (File, error)
	Update(ctx context.Context, userID int64, req UpdateRequest) (File, error)
	Delete(ctx context.Context, id, projectID, userID int64) error
}

type CreateRequest struct {
	Name        string
	ProjectID   int64
	UploadName  string
	ContentType string
	Size        int64
	Body        io.Reader
}

type UpdateRequest struct {
	ID          int64
	ProjectID   int64
	Name        string
	UploadName  string
	ContentType string
	Size        int64
	Body        io.Reader
}

type service struct {
	repo    Repository
	storage Storage
}

func NewService(repo Repository, storage Storage) Service {
	return &service{repo: repo, storage: storage}
}

func (s *service) Create(ctx context.Context, userID int64, req CreateRequest) (File, error) {
	key, url, err := s.storage.Upload(ctx, req.ProjectID, req.UploadName, req.Body, req.Size, req.ContentType)
	if err != nil {
		return File{}, err
	}

	created, err := s.repo.Create(ctx, userID, File{
		Name:      req.Name,
		S3URL:     url,
		S3Key:     key,
		ProjectID: req.ProjectID,
	})
	if err != nil {
		_ = s.storage.Delete(ctx, key)
		return File{}, err
	}

	return created, nil
}

func (s *service) ListByProject(ctx context.Context, projectID, userID int64) ([]File, error) {
	return s.repo.ListByProject(ctx, projectID, userID)
}

func (s *service) GetByID(ctx context.Context, id, projectID, userID int64) (File, error) {
	return s.repo.GetByID(ctx, id, projectID, userID)
}

func (s *service) Update(ctx context.Context, userID int64, req UpdateRequest) (File, error) {
	current, err := s.repo.GetByID(ctx, req.ID, req.ProjectID, userID)
	if err != nil {
		return File{}, err
	}

	updated := File{
		ID:        req.ID,
		Name:      req.Name,
		S3URL:     current.S3URL,
		S3Key:     current.S3Key,
		ProjectID: current.ProjectID,
	}

	hasNewObject := req.Body != nil
	var newObjectKey string
	if hasNewObject {
		key, url, err := s.storage.Upload(ctx, req.ProjectID, req.UploadName, req.Body, req.Size, req.ContentType)
		if err != nil {
			return File{}, err
		}
		updated.S3URL = url
		updated.S3Key = key
		newObjectKey = key
	}

	result, err := s.repo.Update(ctx, userID, updated)
	if err != nil {
		if newObjectKey != "" {
			_ = s.storage.Delete(ctx, newObjectKey)
		}
		return File{}, err
	}

	if hasNewObject && current.S3Key != "" && current.S3Key != result.S3Key {
		if err := s.storage.Delete(ctx, current.S3Key); err != nil {
			return File{}, err
		}
	}

	return result, nil
}

func (s *service) Delete(ctx context.Context, id, projectID, userID int64) error {
	current, err := s.repo.GetByID(ctx, id, projectID, userID)
	if err != nil {
		return err
	}

	if current.S3Key != "" {
		if err := s.storage.Delete(ctx, current.S3Key); err != nil {
			return err
		}
	}

	err = s.repo.Delete(ctx, id, projectID, userID)
	if err != nil {
		return err
	}

	return nil
}

var ErrInvalidMultipart = errors.New("invalid multipart request")
