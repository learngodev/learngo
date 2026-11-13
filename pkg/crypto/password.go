package crypto

import "golang.org/x/crypto/bcrypt"

// HashPassword 使用 bcrypt 对明文密码进行哈希。
func HashPassword(raw string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ComparePassword 比较哈希密码与输入的明文密码。
func ComparePassword(hashed, candidate string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(candidate))
}
