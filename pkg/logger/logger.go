package logger

import (
	"log"
	"os"
)

// Logger 封装标准库 logger，便于日后替换。
type Logger struct {
	*log.Logger
}

// New 创建已配置的 logger 实例。
func New() *Logger {
	return &Logger{Logger: log.New(os.Stdout, "[learn-go] ", log.LstdFlags|log.Lshortfile)}
}
