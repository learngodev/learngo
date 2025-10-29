package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"learn-go/internal/config"
	"learn-go/internal/domain"
	"learn-go/internal/repository"
	"learn-go/pkg/crypto"
)

// AuthService handles authentication and token issuance.
type AuthService struct {
	accounts repository.AccountRepository
	cfg      config.AppConfig
}

var (
	// ErrAccountLocked indicates the account is locked by an administrator.
	ErrAccountLocked = errors.New("account locked")
	// ErrPasswordResetRequired indicates the account must reset password before login.
	ErrPasswordResetRequired = errors.New("password reset required")
	// ErrInvalidCredentials indicates either the username or password is incorrect.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// NewAuthService creates a new AuthService.
func NewAuthService(accounts repository.AccountRepository, cfg config.AppConfig) *AuthService {
	return &AuthService{accounts: accounts, cfg: cfg}
}

// Login authenticates a user and returns JWT access and refresh tokens.
func (s *AuthService) Login(ctx context.Context, schoolID, identifier, password string) (string, string, *domain.Account, error) {
	account, err := s.accounts.FindByIdentifier(ctx, schoolID, identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", nil, ErrInvalidCredentials
		}
		return "", "", nil, err
	}

	switch account.Status {
	case domain.AccountStatusLocked:
		return "", "", nil, ErrAccountLocked
	case domain.AccountStatusPasswordResetRequired:
		return "", "", nil, ErrPasswordResetRequired
	}
	if err := crypto.ComparePassword(account.PasswordHash, password); err != nil {
		return "", "", nil, ErrInvalidCredentials
	}

	accessToken, err := s.generateToken(account.ID, string(account.Role), s.cfg.JWTSecret, s.cfg.TokenTTL)
	if err != nil {
		return "", "", nil, err
	}

	refreshToken, err := s.generateToken(account.ID, string(account.Role), s.cfg.RefreshSecret, s.cfg.RefreshTokenTTL)
	if err != nil {
		return "", "", nil, err
	}

	return accessToken, refreshToken, account, nil
}

func (s *AuthService) generateToken(subjectID, role, secret string, ttlSeconds int64) (string, error) {
	claims := jwt.MapClaims{
		"sub":  subjectID,
		"role": role,
		"exp":  time.Now().Add(time.Duration(ttlSeconds) * time.Second).Unix(),
		"iat":  time.Now().Unix(),
		"jti":  uuid.NewString(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
