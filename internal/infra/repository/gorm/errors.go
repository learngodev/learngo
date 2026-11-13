package gormrepo

import (
	"errors"

	"gorm.io/gorm"

	"learn-go/internal/domain"
)

// notFound 将 gorm.ErrRecordNotFound 转换为 domain.ErrNotFound，其他错误原样返回。
func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}
	return err
}
