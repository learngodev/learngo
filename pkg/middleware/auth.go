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

// Context 键。
const (
	ContextAccountID = "accountID"
	ContextRole      = "role"
	ContextSchoolID  = "schoolID"
)

// AuthConfig 保存用于校验 JWT 的密钥。
type AuthConfig struct {
	Secret       string
	AllowedRoles []string
}

// JWTAuth 返回一个校验 JWT 并可选地限制角色的中间件。
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

		accountID, role, schoolID, err := ValidateJWT(tokenString, cfg.Secret)
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
		c.Set(ContextSchoolID, schoolID)
		c.Next()
	}
}

// ValidateJWT 解析并校验 JWT，成功时返回账号 ID、角色和学校 ID。
func ValidateJWT(tokenString, secret string) (string, string, string, error) {
	if strings.TrimSpace(tokenString) == "" {
		return "", "", "", errInvalidToken
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", "", "", errInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", "", errInvalidTokenClaims
	}

	accountID, _ := claims["sub"].(string)
	role, _ := claims["role"].(string)
	schoolID, _ := claims["school_id"].(string)
	if accountID == "" {
		return "", "", "", errInvalidTokenSub
	}

	return accountID, role, schoolID, nil
}
