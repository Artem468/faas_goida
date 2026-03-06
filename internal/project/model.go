package project

import "time"

type Project struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	UserID    int64     `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}
