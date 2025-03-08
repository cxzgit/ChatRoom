package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"goEnv/gocode/RedisExam/chatRoom/proto"
	"goEnv/gocode/RedisExam/chatRoom/server/utils"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Message struct {
	System   bool
	Username string
	Content  string
}

func NewMessage(system bool, username, content string) Message {
	return Message{System: system, Username: username, Content: content}
}

func (m Message) ToMap() map[string]string {
	return map[string]string{
		"system":   fmt.Sprintf("%t", m.System),
		"username": m.Username,
		"message":  m.Content,
	}
}

var (
	clients *utils.ClientManager

	config *utils.Config
	//心跳超时时间
	heartbeatTimeout time.Duration
	//心跳间隔时间
	heartbeatInterval time.Duration
	//声明一个Redis客户端指针
	rdb *utils.RedisClient
	// 全局日志记录器
	logger *log.Logger
)

// 在程序启动时初始化Redis客户端连接
func init() {
	var err error
	config, err = utils.LoadConfig()
	// 加载配置文件
	if err != nil {
		log.Println("加载配置文件失败:", err)
	}
	// 将心跳超时和检测间隔转换为 time.Duration（单位：秒）
	heartbeatTimeout = time.Duration(config.HeartbeatTimeout) * time.Second
	heartbeatInterval = time.Duration(config.HeartbeatInterval) * time.Second
	// 初始化 Redis 客户端
	rdb = utils.NewRedisClient(config)
	// 初始化日志文件
	logger, err = utils.InitLogger(config.LogFile)
	if err != nil {
		fmt.Println("日志文件初始化失败：", err)
	}

	clients = utils.NewClientManager(logger)
}

// process 处理单个客户端连接
func process(conn net.Conn) {

	var client *utils.ClientInfo
	defer conn.Close()
	reader := bufio.NewReader(conn)

	var name string
	// 循环读取用户名，确保用户名不重复
	for {

		recvName, err := proto.Decode(reader)
		if err != nil {
			logger.Println("读取用户名错误:", err)
			return
		}

		// 去除用户名前后的空白字符
		trimmedName := strings.TrimSpace(recvName)

		// 检查用户名是否为空或仅包含空白字符
		if trimmedName == "" {
			errMsg, _ := proto.Encode("用户名不能为空，请重新输入")
			conn.Write(errMsg)
			continue
		}

		// 检查是否重复
		duplicate := clients.IsExistUserName(trimmedName)

		if duplicate {
			// 发送错误消息，请求重新输入
			errMsg, _ := proto.Encode("用户名已存在，请重新输入")
			conn.Write(errMsg)

			logger.Println("拒绝重复用户名:", recvName)
			continue
		} else {
			name = recvName
			// 发送欢迎消息作为用户名确认
			welcomeMsg, _ := proto.Encode("欢迎" + name + " 进入聊天室~")
			conn.Write(welcomeMsg)
			//创建新用户
			client = utils.NewClient(conn, name)
			break
		}
	}

	// 保存客户端信息
	clients.AddClient(conn, client)

	logger.Println("用户名接收成功:", name)

	//添加用户到排行榜
	rdb.IncrUserActivity(name)
	welMes := "欢迎 " + name + " 进入聊天室"
	messageWel := NewMessage(true, "SYSTEM", welMes)
	welcomeData := messageWel.ToMap()
	if err := rdb.PushMessage("chat_queue_sys", welcomeData); err != nil {
		logger.Println("欢迎消息入队失败：", err)
	}

	// 循环接收客户端消息
	for {
		msg, err := proto.Decode(reader)
		if err != nil {
			if err == io.EOF {
				logger.Println("客户端断开连接:", name)
			} else {

				logger.Println("读取消息错误:", err)
			}
			clients.RemoveClient(conn)
			//当客户端断开连接或者读取消息的时候报错的话，将该用户的活跃度删除
			rdb.RemoveUser(name)
			return
		}

		// 如果收到心跳消息 "ping"，更新最后心跳时间，不进行广播
		if msg == "/PING" {
			client.LastHeartbeat = time.Now()

			fmt.Printf("更新完时间：%s,更新时间:%s\n", client.LastHeartbeat, time.Now())
			continue
		}

		// 处理退出指令
		if msg == "/EXIT" {
			quitMes := name + " 退出聊天室"
			messageQuit := NewMessage(true, "SYSTEM", quitMes)
			quitData := messageQuit.ToMap()
			logger.Println(name, "退出群聊")

			if err := rdb.PushMessage("chat_queue_sys", quitData); err != nil {

				logger.Println("退出消息入队失败：", err)
			}

			//从map集合中删除该用户
			clients.RemoveClient(conn)
			//将该用户的活跃度删除
			rdb.RemoveUser(name)
			return
		}
		if msg == "/RANK" {
			rank := displayRank()
			rankMsg, _ := proto.Encode(rank)
			conn.Write(rankMsg)
			continue
		}

		norMes := msg
		messageNor := NewMessage(false, name, norMes)
		messageDate := messageNor.ToMap()
		//存入Redis队列中
		if err := rdb.PushMessage("chat_queue_nor", messageDate); err != nil {

			logger.Println("消息入队失败：", err)
		}

	}
}

func displayRank() string {
	// 获取用户排名和分数
	result, err := rdb.ShowRange()
	if err != nil {
		return fmt.Sprintf("rdb.ZRevRangeWithScores err:", err)
	}

	if len(result) == 0 {
		return fmt.Sprintf("当前没有用户的活跃度数据哦！🌱")
	}

	// 排行榜消息
	rankMessage := "        🏆 **聊天室活跃度排行榜** 🏆\n"

	for k, v := range result {
		rank := k + 1
		user := v.Member.(string)
		score := v.Score

		// 添加不同排名的称号
		var rankTitle string
		switch rank {
		case 1:
			rankTitle = "🔥 超级活跃王"
		case 2:
			rankTitle = "⚡ 闪耀全场"
		case 3:
			rankTitle = "🌟 聊天小达人"
		default:
			rankTitle = "💬 活跃用户"
		}

		// 添加emoji🏆🥈🥉
		emoji := "🎖️"
		if rank == 1 {
			emoji = "🥇"
		} else if rank == 2 {
			emoji = "🥈"
		} else if rank == 3 {
			emoji = "🥉"
		}
		// 组装消息
		rankMessage += fmt.Sprintf("%s 第%d名: ** %s ** (%s) - 活跃度: %.0f 分\n", emoji, rank, user, rankTitle, score)
	}
	return rankMessage
}

// heartbeatChecker 定时检查客户端心跳
func heartbeatChecker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Println("心跳检测 goroutine 停止")
			return
		case <-ticker.C:
			clients.HeartbeatChecker(heartbeatTimeout)
		}
	}
}

// redisBroadcast 从 Redis 队列中取出系统消息并广播给所有在线客户端
func redisBroadcastSysMes(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		// BRPop 阻塞等待消息，timeout 0 表示无限等待

		sysMsg, err := rdb.PopMessage(ctx, "chat_queue_sys")

		if err != nil {
			// 如果是由于 ctx 取消导致的错误，也可以直接返回
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				logger.Println("系统消息广播 goroutine停止:", err)
				return
			}
			if err == redis.Nil { // redis.Nil 表示没有消息
				// 可以选择直接 continue，不记录错误日志
				continue
			}
			logger.Println("BRPop 错误:", err)
			continue
		}
		sysMessage := sysMsg["message"]

		encodedSysMsg, _ := proto.Encode(sysMessage)
		//广播系统消息
		clients.BroadcastSysMessage(encodedSysMsg)

	}
}

// redisBroadcast 从 Redis 队列中取出普通消息并广播给其他在线客户端
func redisBroadcastNorMsg(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		// BRPop 阻塞等待消息，timeout 0 表示无限等待
		messageData, err := rdb.PopMessage(ctx, "chat_queue_nor")
		if err != nil {
			// 如果是由于 ctx 取消导致的错误，也可以直接返回
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				logger.Println("普通消息广播 goroutine停止:", err)
				return
			}
			if err == redis.Nil { // redis.Nil 表示没有消息
				// 可以选择直接 continue，不记录错误日志
				continue
			}
			logger.Println("BRPop 错误:", err)
			continue
		}
		//获取用户名和消息
		name := messageData["username"]
		msg := messageData["message"]

		logger.Println("发送者:", name, "消息:", msg)

		//广播消息
		//将消息进行编码
		encodedMsg, _ := proto.Encode(fmt.Sprintf("%s:%s", name, msg))
		//广播普通消息
		clients.BroadcastNorMessage(encodedMsg, name)

		//用户每发送一条消息活跃度加一
		rdb.IncrUserActivity(name)
	}
}

// 服务器安全退出进行资源清理
func shutdownServer() {

	logger.Println("执行服务器安全退出...")

	// 1. 通知所有客户端断开连接
	encodedMsg, _ := proto.Encode("服务器即将关闭，所有连接将断开...")
	//广播给所有用户
	clients.BroadcastSysMessage(encodedMsg)

	// 2. 清理 Redis 相关数据
	// 删除活跃度排行榜
	rdb.Del("chatRoom_user_activity")
	// 清空系统消息队列
	rdb.Del("chat_queue_sys")
	// 清空普通消息队列
	rdb.Del("chat_queue_nor")

	// 3. 关闭 Redis 连接
	rdb.Close()

	logger.Println("服务器已安全退出。")
	os.Exit(0)
}

func main() {
	//创建上下文和waitGroup用于优雅的关闭goroutine
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	// 监听地址采用配置文件中的端口
	listenAddr := ":" + config.ServerPort
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		logger.Println("监听失败:", err)
		return
	}
	defer listener.Close()

	logger.Println("服务器已启动，监听端口:", config.ServerPort)

	// 启动心跳检查 goroutine
	wg.Add(1)
	go heartbeatChecker(ctx, &wg)

	// 启动 Redis 队列消息消费与广播  欢迎和退出消息
	wg.Add(1)
	go redisBroadcastSysMes(ctx, &wg)

	//启动 Redis 队列消息广播 其他客户端获取其他用户的消息
	wg.Add(1)
	go redisBroadcastNorMsg(ctx, &wg)

	// 监听 OS 终止信号
	signalChan := make(chan os.Signal, 1)

	//主要用于监听 操作系统的信号（signals），
	//特别是 SIGINT 和 SIGTERM，从而实现 优雅退出 机制。
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					//收到退出信号后退出循环
					return
				default:
					logger.Println("接受连接错误:", err)
					continue
				}

			}
			logger.Printf("新客户端连接: %v\n", conn.RemoteAddr().String())
			go process(conn)
		}
	}()
	// 等待终止信号
	<-signalChan
	logger.Println("服务器即将关闭...")

	//通知所有goroutine退出

	cancel()

	//关闭listener，防止连接阻塞
	listener.Close()
	//等待所有goroutine正常退出
	wg.Wait()
	// 调用安全退出函数
	shutdownServer()
}
