package utils

import (
	"net"
	"sync"
	"time"
)

// ClientInfo 存储客户端信息
type ClientInfo struct {
	Name          string
	Conn          net.Conn
	LoginTime     time.Time
	LastHeartbeat time.Time
	HbMutex       sync.Mutex
	WriteMutex    sync.Mutex
}

// NewClient 创建新客户端
func NewClient(conn net.Conn, name string) *ClientInfo {
	return &ClientInfo{
		Name:          name,
		Conn:          conn,
		LoginTime:     time.Now(),
		LastHeartbeat: time.Now(),
	}
}
