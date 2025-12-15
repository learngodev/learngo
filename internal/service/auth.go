package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
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
	accounts    repository.AccountRepository
	resetTokens repository.PasswordResetTokenRepository
	cfg         config.AppConfig
}

var (
	// ErrAccountLocked indicates the account is locked by an administrator.
	ErrAccountLocked = errors.New("account locked")
	// ErrPasswordResetRequired indicates the account must reset password before login.
	ErrPasswordResetRequired = errors.New("password reset required")
	// ErrInvalidCredentials indicates either the username or password is incorrect.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidRefreshToken indicates refresh token parsing/validation failure.
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	// ErrPasswordResetTokenInvalid indicates the provided token is not usable.
	ErrPasswordResetTokenInvalid = errors.New("password reset token invalid")
	// ErrPasswordResetTokenExpired indicates token expired.
	ErrPasswordResetTokenExpired = errors.New("password reset token expired")
	// ErrPasswordResetUnavailable indicates reset flow cannot be used at the moment.
	ErrPasswordResetUnavailable = errors.New("password reset unavailable")
)

// NewAuthService creates a new AuthService.
func NewAuthService(accounts repository.AccountRepository, tokens repository.PasswordResetTokenRepository, cfg config.AppConfig) *AuthService {
	return &AuthService{accounts: accounts, resetTokens: tokens, cfg: cfg}
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

// RefreshTokens validates refresh token and issues a new pair.
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (string, string, *domain.Account, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return "", "", nil, ErrInvalidRefreshToken
	}

	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %T", token.Method)
		}
		return []byte(s.cfg.RefreshSecret), nil
	})
	if err != nil || !parsed.Valid {
		return "", "", nil, ErrInvalidRefreshToken
	}

	subject, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)
	if subject == "" || role == "" {
		return "", "", nil, ErrInvalidRefreshToken
	}

	account, err := s.accounts.FindByID(ctx, subject)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", nil, ErrInvalidRefreshToken
		}
		return "", "", nil, err
	}

	switch account.Status {
	case domain.AccountStatusLocked:
		return "", "", nil, ErrAccountLocked
	case domain.AccountStatusPasswordResetRequired:
		return "", "", nil, ErrPasswordResetRequired
	}

	accessToken, err := s.generateToken(account.ID, string(account.Role), s.cfg.JWTSecret, s.cfg.TokenTTL)
	if err != nil {
		return "", "", nil, err
	}
	newRefresh, err := s.generateToken(account.ID, string(account.Role), s.cfg.RefreshSecret, s.cfg.RefreshTokenTTL)
	if err != nil {
		return "", "", nil, err
	}

	return accessToken, newRefresh, account, nil
}

// GetAccount retrieves an account by ID.
func (s *AuthService) GetAccount(ctx context.Context, accountID string) (*domain.Account, error) {
	return s.accounts.FindByID(ctx, accountID)
}

// RequestPasswordReset creates a single-use token and returns plain value plus expiry.
func (s *AuthService) RequestPasswordReset(ctx context.Context, schoolID, identifier string) (string, time.Time, error) {
	if s.resetTokens == nil {
		return "", time.Time{}, ErrPasswordResetUnavailable
	}
	if s.cfg.PasswordResetTokenTTL <= 0 {
		return "", time.Time{}, ErrPasswordResetUnavailable
	}

	account, err := s.accounts.FindByIdentifier(ctx, schoolID, identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", time.Time{}, ErrInvalidCredentials
		}
		return "", time.Time{}, err
	}

	if account.Status != domain.AccountStatusPasswordResetRequired {
		return "", time.Time{}, ErrPasswordResetUnavailable
	}

	if err := s.resetTokens.DeleteByAccount(ctx, account.ID); err != nil {
		return "", time.Time{}, err
	}

	plain, err := generateSecureToken()
	if err != nil {
		return "", time.Time{}, err
	}
	hash := hashToken(plain)
	expiresAt := time.Now().Add(time.Duration(s.cfg.PasswordResetTokenTTL) * time.Second)

	token := &domain.PasswordResetToken{
		ID:        uuid.NewString(),
		AccountID: account.ID,
		SchoolID:  schoolID,
		TokenHash: hash,
		ExpiresAt: expiresAt,
	}

	if err := s.resetTokens.Create(ctx, token); err != nil {
		return "", time.Time{}, err
	}

	return plain, expiresAt, nil
}

// ResetPassword validates token and updates password hash.
func (s *AuthService) ResetPassword(ctx context.Context, schoolID, identifier, token, newPassword string) error {
	if s.resetTokens == nil {
		return ErrPasswordResetUnavailable
	}
	if strings.TrimSpace(token) == "" || strings.TrimSpace(newPassword) == "" {
		return ErrPasswordResetTokenInvalid
	}

	account, err := s.accounts.FindByIdentifier(ctx, schoolID, identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidCredentials
		}
		return err
	}
	if account.Status != domain.AccountStatusPasswordResetRequired {
		return ErrPasswordResetUnavailable
	}

	hash := hashToken(token)
	record, err := s.resetTokens.FindByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPasswordResetTokenInvalid
		}
		return err
	}
	if record.AccountID != account.ID {
		return ErrPasswordResetTokenInvalid
	}
	if record.ConsumedAt != nil {
		return ErrPasswordResetTokenInvalid
	}
	if time.Now().After(record.ExpiresAt) {
		return ErrPasswordResetTokenExpired
	}

	passwordHash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.accounts.UpdatePasswordHash(ctx, account.ID, passwordHash); err != nil {
		return err
	}
	if err := s.accounts.UpdateStatus(ctx, account.ID, schoolID, domain.AccountStatusActive); err != nil {
		return err
	}
	if err := s.resetTokens.Consume(ctx, record.ID, time.Now()); err != nil {
		return err
	}
	return nil
}

func generateSecureToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
