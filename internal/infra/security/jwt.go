package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"learn-go/internal/domain/identity"
)

// ErrInvalidToken 表示令牌解析或校验失败。
var ErrInvalidToken = errors.New("invalid token")

// JWTIssuer 基于 HS256 JWT 实现 identity.TokenIssuer。
type JWTIssuer struct {
	accessSecret  string
	refreshSecret string
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

// NewJWTIssuer 创建 JWTIssuer，TTL 单位为秒。
func NewJWTIssuer(accessSecret, refreshSecret string, accessTTLSeconds, refreshTTLSeconds int64) *JWTIssuer {
	return &JWTIssuer{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		accessTTL:     time.Duration(accessTTLSeconds) * time.Second,
		refreshTTL:    time.Duration(refreshTTLSeconds) * time.Second,
	}
}

func (j *JWTIssuer) IssueAccessToken(claims identity.TokenClaims) (string, error) {
	return sign(claims, j.accessSecret, j.accessTTL)
}

func (j *JWTIssuer) IssueRefreshToken(claims identity.TokenClaims) (string, error) {
	return sign(claims, j.refreshSecret, j.refreshTTL)
}

func (j *JWTIssuer) ParseRefreshToken(token string) (*identity.TokenClaims, error) {
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %T", token.Method)
		}
		return []byte(j.refreshSecret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	subject, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)
	schoolID, _ := claims["school_id"].(string)
	return &identity.TokenClaims{AccountID: subject, Role: identity.Role(role), SchoolID: schoolID}, nil
}

func sign(claims identity.TokenClaims, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":       claims.AccountID,
		"role":      string(claims.Role),
		"school_id": claims.SchoolID,
		"exp":       now.Add(ttl).Unix(),
		"iat":       now.Unix(),
		"jti":       uuid.NewString(),
	})
	return token.SignedString([]byte(secret))
}

var _ identity.TokenIssuer = (*JWTIssuer)(nil)
