package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"faas_goida/internal/user"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRefresh     = errors.New("invalid refresh token")
	ErrInvalidInput       = errors.New("invalid input")
)

type Service interface {
	Register(ctx context.Context, email, password string) (user.User, error)
	Login(ctx context.Context, email, password string) (TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (TokenPair, error)
}

type service struct {
	users  user.Repository
	tokens RefreshTokenRepository
	cfg    ServiceConfig
	now    func() time.Time
}

func NewService(users user.Repository, tokens RefreshTokenRepository, cfg ServiceConfig) Service {
	return &service{
		users:  users,
		tokens: tokens,
		cfg:    cfg,
		now:    time.Now,
	}
}

func (s *service) Register(ctx context.Context, email, password string) (user.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validEmail(email) || !validPassword(password) {
		return user.User{}, ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return user.User{}, err
	}

	return s.users.Create(ctx, user.User{
		Email:        email,
		PasswordHash: string(hash),
	})
}

func (s *service) Login(ctx context.Context, email, password string) (TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validEmail(email) || password == "" {
		return TokenPair{}, ErrInvalidCredentials
	}

	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			return TokenPair{}, ErrInvalidCredentials
		}
		return TokenPair{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, u.ID)
}

func (s *service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return TokenPair{}, ErrInvalidRefresh
	}

	tokenHash := hashToken(refreshToken)
	stored, err := s.tokens.Get(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return TokenPair{}, ErrInvalidRefresh
		}
		return TokenPair{}, err
	}

	now := s.now()
	if stored.ExpiresAt.Before(now) {
		_ = s.tokens.Delete(ctx, tokenHash)
		return TokenPair{}, ErrInvalidRefresh
	}

	if err := s.tokens.Delete(ctx, tokenHash); err != nil {
		return TokenPair{}, err
	}

	return s.issueTokens(ctx, stored.UserID)
}

func (s *service) issueTokens(ctx context.Context, userID int64) (TokenPair, error) {
	refreshToken, err := generateToken(48)
	if err != nil {
		return TokenPair{}, err
	}

	now := s.now()
	accessExp := now.Add(s.cfg.AccessTTL)
	refreshExp := now.Add(s.cfg.RefreshTTL)

	if s.cfg.AccessSecret == "" {
		return TokenPair{}, errors.New("access token secret is required")
	}

	accessToken, err := generateAccessToken(userID, s.cfg.AccessSecret, accessExp)
	if err != nil {
		return TokenPair{}, err
	}
	refreshHash := hashToken(refreshToken)

	if err := s.tokens.Store(ctx, userID, refreshHash, refreshExp); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccessExp:    accessExp,
		RefreshExp:   refreshExp,
	}, nil
}
