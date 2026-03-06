package file

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrFileNotFound      = errors.New("file not found")
	ErrProjectNotFoundFK = errors.New("project not found")
)

type Repository interface {
	Create(ctx context.Context, userID int64, f File) (File, error)
	ListByProject(ctx context.Context, projectID, userID int64) ([]File, error)
	GetByID(ctx context.Context, id, projectID, userID int64) (File, error)
	Update(ctx context.Context, userID int64, f File) (File, error)
	Delete(ctx context.Context, id, projectID, userID int64) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, userID int64, f File) (File, error) {
	query := `
		INSERT INTO files (name, s3_url, s3_key, project_id)
		SELECT $1, $2, $3, p.id
		FROM projects p
		WHERE p.id = $4 AND p.user_id = $5
		RETURNING id
	`
	err := r.db.QueryRowContext(ctx, query, f.Name, f.S3URL, f.S3Key, f.ProjectID, userID).Scan(&f.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return File{}, ErrProjectNotFoundFK
		}
		return File{}, err
	}
	return f, nil
}

func (r *repository) ListByProject(ctx context.Context, projectID, userID int64) ([]File, error) {
	query := `
		SELECT f.id, f.name, f.s3_url, f.s3_key, f.project_id
		FROM files f
		INNER JOIN projects p ON p.id = f.project_id
		WHERE f.project_id = $1 AND p.user_id = $2
		ORDER BY f.id
	`

	rows, err := r.db.QueryContext(ctx, query, projectID, userID)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			return
		}
	}(rows)

	out := make([]File, 0)
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.Name, &f.S3URL, &f.S3Key, &f.ProjectID); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *repository) GetByID(ctx context.Context, id, projectID, userID int64) (File, error) {
	query := `
		SELECT f.id, f.name, f.s3_url, f.s3_key, f.project_id
		FROM files f
		INNER JOIN projects p ON p.id = f.project_id
		WHERE f.id = $1 AND f.project_id = $2 AND p.user_id = $3
	`

	var f File
	err := r.db.QueryRowContext(ctx, query, id, projectID, userID).Scan(&f.ID, &f.Name, &f.S3URL, &f.S3Key, &f.ProjectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return File{}, ErrFileNotFound
		}
		return File{}, err
	}
	return f, nil
}

func (r *repository) Update(ctx context.Context, userID int64, f File) (File, error) {
	query := `
		UPDATE files
		SET name = $1, s3_url = $2, s3_key = $3
		WHERE id = $4
		AND project_id = $5
		AND EXISTS (
			SELECT 1
			FROM projects p
			WHERE p.id = files.project_id AND p.user_id = $6
		)
		RETURNING id
	`

	err := r.db.QueryRowContext(ctx, query, f.Name, f.S3URL, f.S3Key, f.ID, f.ProjectID, userID).Scan(&f.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return File{}, ErrFileNotFound
		}
		return File{}, err
	}
	return f, nil
}

func (r *repository) Delete(ctx context.Context, id, projectID, userID int64) error {
	query := `
		DELETE FROM files f
		USING projects p
		WHERE f.id = $1
		AND f.project_id = $2
		AND p.id = f.project_id
		AND p.user_id = $3
	`
	res, err := r.db.ExecContext(ctx, query, id, projectID, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrFileNotFound
	}
	return nil
}
