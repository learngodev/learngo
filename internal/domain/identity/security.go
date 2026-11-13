package identity

// PasswordHasher 负责密码的哈希与校验。
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
}

// TokenClaims 是登录令牌中携带的身份信息。
type TokenClaims struct {
	AccountID string
	Role      Role
	SchoolID  string
}

// TokenIssuer 负责签发访问令牌/刷新令牌，并解析刷新令牌。
type TokenIssuer interface {
	IssueAccessToken(claims TokenClaims) (string, error)
	IssueRefreshToken(claims TokenClaims) (string, error)
	ParseRefreshToken(token string) (*TokenClaims, error)
}
