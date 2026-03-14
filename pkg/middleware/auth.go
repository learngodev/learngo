package middleware

import (
	"errors"
	"net/http"
	"strings"

	"learn-go/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	errInvalidToken       = errors.New(response.CodeInvalidToken)
	errInvalidTokenClaims = errors.New(response.CodeInvalidTokenClaims)
	errInvalidTokenSub    = errors.New(response.CodeInvalidTokenSubject)
)

// Context keys.
const (
	ContextAccountID = "accountID"
	ContextRole      = "role"
)

// AuthConfig holds secret used to verify JWT.
type AuthConfig struct {
	Secret       string
	AllowedRoles []string
}

// JWTAuth returns a middleware that verifies JWT tokens and optionally enforces roles.
func JWTAuth(cfg AuthConfig) gin.HandlerFunc {
	roleSet := make(map[string]struct{}, len(cfg.AllowedRoles))
	for _, role := range cfg.AllowedRoles {
		roleSet[role] = struct{}{}
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			response.Error(c, http.StatusUnauthorized, response.CodeMissingAuthorization, nil)
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(authHeader[7:])

		accountID, role, err := ValidateJWT(tokenString, cfg.Secret)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidTokenClaims):
				response.Error(c, http.StatusUnauthorized, response.CodeInvalidTokenClaims, nil)
			case errors.Is(err, errInvalidTokenSub):
				response.Error(c, http.StatusUnauthorized, response.CodeInvalidTokenSubject, nil)
			default:
				response.Error(c, http.StatusUnauthorized, response.CodeInvalidToken, nil)
			}
			c.Abort()
			return
		}

		if len(roleSet) > 0 {
			if _, ok := roleSet[role]; !ok {
				response.Error(c, http.StatusForbidden, response.CodeInsufficientRole, nil)
				c.Abort()
				return
			}
		}

		c.Set(ContextAccountID, accountID)
		c.Set(ContextRole, role)
		c.Next()
	}
}

// ValidateJWT parses and validates the JWT token, returning account ID and role if successful.
func ValidateJWT(tokenString, secret string) (string, string, error) {
	if strings.TrimSpace(tokenString) == "" {
		return "", "", errInvalidToken
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", "", errInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errInvalidTokenClaims
	}

	accountID, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)
	if accountID == "" {
		return "", "", errInvalidTokenSub
	}

	return accountID, role, nil
}
