package domain

import "errors"

// ErrNotFound 表示仓储查询未找到记录。
var ErrNotFound = errors.New("repository: not found")
