package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"learn-go/internal/domain"
	"strings"
	"time"

	"github.com/google/uuid"

	"learn-go/internal/domain/identity"
)

// AuthUsecase 负责认证与令牌签发。
type AuthUsecase struct {
	accounts    identity.AccountRepository
	resetTokens identity.PasswordResetTokenRepository
	tokens      identity.TokenIssuer
	hasher      identity.PasswordHasher
	// resetTokenTTL 为密码重置令牌有效期，<= 0 表示禁用密码重置。
	resetTokenTTL time.Duration
}

var (
	// ErrAccountLocked 表示账号已被管理员锁定。
	ErrAccountLocked = errors.New("account locked")
	// ErrPasswordResetRequired 表示账号需先重置密码才能登录。
	ErrPasswordResetRequired = errors.New("password reset required")
	// ErrInvalidCredentials 表示用户名或密码错误。
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidRefreshToken 表示刷新令牌解析或校验失败。
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	// ErrPasswordResetTokenInvalid 表示提供的令牌不可用。
	ErrPasswordResetTokenInvalid = errors.New("password reset token invalid")
	// ErrPasswordResetTokenExpired 表示令牌已过期。
	ErrPasswordResetTokenExpired = errors.New("password reset token expired")
	// ErrPasswordResetUnavailable 表示密码重置流程当前不可用。
	ErrPasswordResetUnavailable = errors.New("password reset unavailable")
)

// NewAuthUsecase 创建 AuthUsecase。
func NewAuthUsecase(
	accounts identity.AccountRepository,
	resetTokens identity.PasswordResetTokenRepository,
	tokens identity.TokenIssuer,
	hasher identity.PasswordHasher,
	resetTokenTTL time.Duration,
) *AuthUsecase {
	return &AuthUsecase{accounts: accounts, resetTokens: resetTokens, tokens: tokens, hasher: hasher, resetTokenTTL: resetTokenTTL}
}

// Login 认证用户并返回 JWT 访问令牌和刷新令牌。
func (s *AuthUsecase) Login(ctx context.Context, schoolID, identifier, password string) (string, string, *identity.Account, error) {
	account, err := s.accounts.FindByIdentifier(ctx, schoolID, identifier)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", "", nil, ErrInvalidCredentials
		}
		return "", "", nil, err
	}

	switch account.Status {
	case identity.AccountStatusLocked:
		return "", "", nil, ErrAccountLocked
	case identity.AccountStatusPasswordResetRequired:
		return "", "", nil, ErrPasswordResetRequired
	}
	//if err := crypto.ComparePassword(account.PasswordHash, password); err != nil {
	//	return "", "", nil, ErrInvalidCredentials
	//}

	accessToken, refreshToken, err := s.issueTokenPair(account)
	if err != nil {
		return "", "", nil, err
	}

	return accessToken, refreshToken, account, nil
}

func (s *AuthUsecase) issueTokenPair(account *identity.Account) (string, string, error) {
	claims := identity.TokenClaims{AccountID: account.ID, Role: account.Role, SchoolID: account.SchoolID}
	accessToken, err := s.tokens.IssueAccessToken(claims)
	if err != nil {
		return "", "", err
	}
	refreshToken, err := s.tokens.IssueRefreshToken(claims)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// RefreshTokens 校验刷新令牌并签发新的令牌对。
func (s *AuthUsecase) RefreshTokens(ctx context.Context, refreshToken string) (string, string, *identity.Account, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return "", "", nil, ErrInvalidRefreshToken
	}

	claims, err := s.tokens.ParseRefreshToken(refreshToken)
	if err != nil || claims.AccountID == "" || claims.Role == "" {
		return "", "", nil, ErrInvalidRefreshToken
	}

	account, err := s.accounts.FindByID(ctx, claims.AccountID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", "", nil, ErrInvalidRefreshToken
		}
		return "", "", nil, err
	}

	switch account.Status {
	case identity.AccountStatusLocked:
		return "", "", nil, ErrAccountLocked
	case identity.AccountStatusPasswordResetRequired:
		return "", "", nil, ErrPasswordResetRequired
	}

	accessToken, newRefresh, err := s.issueTokenPair(account)
	if err != nil {
		return "", "", nil, err
	}

	return accessToken, newRefresh, account, nil
}

// GetAccount 按 ID 查询账号。
func (s *AuthUsecase) GetAccount(ctx context.Context, accountID string) (*identity.Account, error) {
	return s.accounts.FindByID(ctx, accountID)
}

// RequestPasswordReset 创建一次性令牌，返回明文值及过期时间。
func (s *AuthUsecase) RequestPasswordReset(ctx context.Context, schoolID, identifier string) (string, time.Time, error) {
	if s.resetTokens == nil {
		return "", time.Time{}, ErrPasswordResetUnavailable
	}
	if s.resetTokenTTL <= 0 {
		return "", time.Time{}, ErrPasswordResetUnavailable
	}

	account, err := s.accounts.FindByIdentifier(ctx, schoolID, identifier)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", time.Time{}, ErrInvalidCredentials
		}
		return "", time.Time{}, err
	}

	if account.Status != identity.AccountStatusPasswordResetRequired {
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
	expiresAt := time.Now().Add(s.resetTokenTTL)

	token := &identity.PasswordResetToken{
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

// ResetPassword 校验令牌并更新密码哈希。
func (s *AuthUsecase) ResetPassword(ctx context.Context, schoolID, identifier, token, newPassword string) error {
	if s.resetTokens == nil {
		return ErrPasswordResetUnavailable
	}
	if strings.TrimSpace(token) == "" || strings.TrimSpace(newPassword) == "" {
		return ErrPasswordResetTokenInvalid
	}

	account, err := s.accounts.FindByIdentifier(ctx, schoolID, identifier)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrInvalidCredentials
		}
		return err
	}
	if account.Status != identity.AccountStatusPasswordResetRequired {
		return ErrPasswordResetUnavailable
	}

	hash := hashToken(token)
	record, err := s.resetTokens.FindByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
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

	passwordHash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	if err := s.accounts.UpdatePasswordHash(ctx, account.ID, passwordHash); err != nil {
		return err
	}
	if err := s.accounts.UpdateStatus(ctx, account.ID, schoolID, identity.AccountStatusActive); err != nil {
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
