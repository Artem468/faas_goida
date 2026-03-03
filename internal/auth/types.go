package auth

import "time"

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	AccessExp    time.Time
	RefreshExp   time.Time
}

type ServiceConfig struct {
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
	AccessSecret string
}

type RefreshToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type registerResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

type tokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	AccessExp    time.Time `json:"access_expires_at"`
	RefreshExp   time.Time `json:"refresh_expires_at"`
}
