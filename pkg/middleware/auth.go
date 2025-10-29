package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	errInvalidToken       = errors.New("invalid token")
	errInvalidTokenClaims = errors.New("invalid token claims")
	errInvalidTokenSub    = errors.New("invalid token subject")
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"message": "missing authorization"}})
			return
		}

		tokenString := strings.TrimSpace(authHeader[7:])

		accountID, role, err := ValidateJWT(tokenString, cfg.Secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"message": err.Error()}})
			return
		}

		if len(roleSet) > 0 {
			if _, ok := roleSet[role]; !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"message": "insufficient role"}})
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
