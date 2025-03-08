package utils

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type ClientManager struct {
	logger  *log.Logger
	clients sync.Map
}

// 初始化
func NewClientManager(logger *log.Logger) *ClientManager {
	return &ClientManager{logger: logger}
}

// 添加新的用户
func (cm *ClientManager) AddClient(conn net.Conn, client *ClientInfo) {
	cm.clients.Store(conn, client)
}

// 删除用户
func (cm *ClientManager) RemoveClient(conn net.Conn) {
	cm.clients.Delete(conn)
}

// 检查用户名是否重复
func (cm *ClientManager) IsExistUserName(name string) bool {
	duplicate := false
	cm.clients.Range(func(key, value interface{}) bool {
		existing := value.(*ClientInfo)
		if name == existing.Name {
			duplicate = true
			return false
		}
		return true
	})
	return duplicate
}

// 广播消息给所有客户端
func (cm *ClientManager) BroadcastSysMessage(msg []byte) {
	cm.clients.Range(func(_, value interface{}) bool {
		client := value.(*ClientInfo)
		//避免消息交叉
		client.WriteMutex.Lock()
		_, err := client.Conn.Write(msg)
		client.WriteMutex.Unlock()
		if err != nil {
			cm.logger.Println("广播系统消息写入错误", err)
		}
		return true
	})
}

// 广播消息给其他用户端
func (cm *ClientManager) BroadcastNorMessage(msg []byte, name string) {
	cm.clients.Range(func(key, value interface{}) bool {
		c := value.(*ClientInfo)
		//发送者不接收消息
		if c.Name != name {
			//避免消息交叉
			c.WriteMutex.Lock()
			_, err := c.Conn.Write(msg)
			c.WriteMutex.Unlock()
			if err != nil {
				cm.logger.Println("广播普通消息写入错误:", err)
			}
		}
		return true
	})
}

// 检查心跳是否超时
func (cm *ClientManager) HeartbeatChecker(heartbeatTimeout time.Duration) {
	now := time.Now()
	cm.clients.Range(func(key, value interface{}) bool {
		client := value.(*ClientInfo)
		//确保读取时不会发生竞争。
		//lastHeartbeat := client.LastHeartbeat
		fmt.Printf("8888用户：%s,时间为%s\n", client.Name, client.LastHeartbeat)
		if now.Sub(client.LastHeartbeat) > heartbeatTimeout {
			fmt.Printf("用户：%s,时间为%s\n", client.Name, client.LastHeartbeat)
			cm.logger.Println("检测到超时，断开连接:", client.Name)
			client.Conn.Close()
			cm.clients.Delete(client.Conn)
		}
		return true
	})
}
