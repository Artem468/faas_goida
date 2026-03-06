package project

import (
	"context"
	"database/sql"
	"errors"
)

var ErrProjectNotFound = errors.New("project not found")

type Repository interface {
	Create(ctx context.Context, p Project) (Project, error)
	List(ctx context.Context, userID int64) ([]Project, error)
	GetByID(ctx context.Context, id, userID int64) (Project, error)
	Update(ctx context.Context, p Project) (Project, error)
	Delete(ctx context.Context, id, userID int64) error
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, p Project) (Project, error) {
	query := `
		INSERT INTO projects (name, user_id)
		VALUES ($1, $2)
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(ctx, query, p.Name, p.UserID).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return Project{}, err
	}
	return p, nil
}

func (r *repository) List(ctx context.Context, userID int64) ([]Project, error) {
	query := `
		SELECT id, name, user_id, created_at
		FROM projects
		WHERE user_id = $1
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			return
		}
	}(rows)

	out := make([]Project, 0)
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.UserID, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *repository) GetByID(ctx context.Context, id, userID int64) (Project, error) {
	query := `
		SELECT id, name, user_id, created_at
		FROM projects
		WHERE id = $1 AND user_id = $2
	`

	var p Project
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(&p.ID, &p.Name, &p.UserID, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Project{}, ErrProjectNotFound
		}
		return Project{}, err
	}
	return p, nil
}

func (r *repository) Update(ctx context.Context, p Project) (Project, error) {
	query := `
		UPDATE projects
		SET name = $1
		WHERE id = $2 AND user_id = $3
		RETURNING created_at
	`

	err := r.db.QueryRowContext(ctx, query, p.Name, p.ID, p.UserID).Scan(&p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Project{}, ErrProjectNotFound
		}
		return Project{}, err
	}
	return p, nil
}

func (r *repository) Delete(ctx context.Context, id, userID int64) error {
	query := `
		DELETE FROM projects
		WHERE id = $1 AND user_id = $2
	`
	res, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrProjectNotFound
	}
	return nil
}
