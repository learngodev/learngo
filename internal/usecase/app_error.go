package usecase

import "fmt"

// AppError 是结构化的业务错误，可安全地映射为 API 响应。
type AppError struct {
	Code    string
	Status  int
	Message string
	Details interface{}
	Cause   error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	if e.Code != "" {
		return e.Code
	}
	return "application error"
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *AppError) String() string {
	if e == nil {
		return ""
	}
	if e.Cause == nil {
		return e.Error()
	}
	return fmt.Sprintf("%s: %v", e.Error(), e.Cause)
}
