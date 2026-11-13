package security

import (
	"learn-go/internal/domain/identity"
	"learn-go/pkg/crypto"
)

// BcryptHasher 基于 bcrypt 实现 identity.PasswordHasher。
type BcryptHasher struct{}

// NewBcryptHasher 创建 BcryptHasher。
func NewBcryptHasher() BcryptHasher { return BcryptHasher{} }

func (BcryptHasher) Hash(plain string) (string, error) { return crypto.HashPassword(plain) }

func (BcryptHasher) Compare(hash, plain string) error { return crypto.ComparePassword(hash, plain) }

var _ identity.PasswordHasher = BcryptHasher{}
