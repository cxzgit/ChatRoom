package utils

import (
	"io"
	"log"
	"os"
)

// InitLogger 初始化日志
func InitLogger(logFile string) (*log.Logger, error) {
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	// 同时输出到标准输出和日志文件
	logger := log.New(io.MultiWriter(os.Stdout, file), "CHAT_SERVER: ", log.Ldate|log.Ltime|log.Lshortfile)
	return logger, nil
}
