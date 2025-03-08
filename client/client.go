package main

import (
	"bufio"
	"context"
	"fmt"
	"goEnv/gocode/RedisExam/chatRoom/client/utils"
	"goEnv/gocode/RedisExam/chatRoom/proto"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	//全局配置
	config *utils.Config
	//声明一个Redis客户端指针
	//心跳间隔时间
	heartbeatInterval time.Duration
)

// 在程序启动时初始化Redis客户端连接
func init() {
	// 1. 加载 config.ini
	var err error
	config, err = utils.LoadConfig()
	// 加载配置文件
	if err != nil {
		log.Println("加载配置文件失败:", err)
		os.Exit(1)
	}
	// 将心跳超时和检测间隔转换为 time.Duration（单位：秒）
	heartbeatInterval = time.Duration(config.HeartbeatInterval) * time.Second
}

func main() {
	//创建上下文和waitGroup用于优雅的关闭goroutine
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	// 拼接服务器地址与端口
	serverAddr := config.ServerAddress + ":" + config.ServerPort
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("net.Dial err:", err)
		return
	}
	defer conn.Close()

	// 用户名握手：循环输入直到服务器确认
	stdinReader := bufio.NewReader(os.Stdin)
	reader := bufio.NewReader(conn)
	for {
		fmt.Print("请输入你的名字: ")
		name, _ := stdinReader.ReadString('\n')
		name = strings.TrimSpace(name)
		encodedName, err := proto.Encode(name)
		if err != nil {
			fmt.Println("proto.Encode err:", err)
			continue
		}

		_, err = conn.Write(encodedName)

		if err != nil {
			fmt.Println("conn.Write err:", err)
			return
		}

		// 等待服务器返回信息
		serverResp, err := proto.Decode(reader)
		if err != nil {
			fmt.Println("等待服务器响应错误:", err)
			return
		}

		// 判断服务器响应内容
		if strings.HasPrefix(serverResp, "用户名已存在") {
			fmt.Println("服务器提示：", serverResp)
			// 继续循环让用户重新输入
			continue
		} else if strings.HasPrefix(serverResp, "用户名不能为空") {
			fmt.Println("服务器提示：", serverResp)
		} else if strings.HasPrefix(serverResp, "欢迎") {
			// 用户名确认通过
			//fmt.Println("服务器提示：", serverResp)
			// 显示界面
			printInterface()
			break
		}
	}

	// 启动一个 goroutine 用于接收服务器消息
	wg.Add(1)
	go recMag(conn, ctx, &wg)

	//启动一个goroutine 用于心跳检测
	wg.Add(1)
	go heartbeat(conn, ctx, &wg)

	// 循环读取用户输入的消息并发送
	for {
		//fmt.Print("请输入消息:")
		input, err := stdinReader.ReadString('\n')
		if err != nil {
			fmt.Println("stdinReader.ReadString err:", err)
			continue
		}
		input = strings.TrimSpace(input)
		//消息不能为空
		if input == "" {
			fmt.Println("消息不能为空")
			continue
		}
		//当用户输入的消息为exit的时候，退出客户端
		if input == "/EXIT" {
			encodedMsg, err := proto.Encode(input)
			if err != nil {
				fmt.Println("proto.Encode err:", err)
			} else {
				_, err = conn.Write(encodedMsg)
				if err != nil {
					fmt.Println("conn.Write err:", err)
				}
			}
			fmt.Println("正在安全退出~")
			cancel()
			wg.Wait()
			fmt.Println("退出~")
			break
		}

		encodedMsg, err := proto.Encode(input)
		if err != nil {
			fmt.Println("proto.Encode err:", err)
			continue
		}
		_, err = conn.Write(encodedMsg)
		if err != nil {
			fmt.Println("conn.Write err:", err)
			continue
		}
	}
}

// 启动一个 goroutine 用于接收服务器消息
func recMag(conn net.Conn, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	reader := bufio.NewReader(conn)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("接收服务器消息关闭")
			return
		default:
			msg, err := proto.Decode(reader)
			if err != nil {
				// 如果是 EOF，则可以认为连接已经关闭，直接退出
				if err == io.EOF {
					return
				}
				fmt.Println("proto.Decode err:", err)
				return
			}
			fmt.Println("收到消息:", msg)
		}
	}
}

// 启动一个goroutine 用于心跳检测
func heartbeat(conn net.Conn, ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	//每隔5s返回一个新的Ticker,包含一个管道字段，每隔5s就向该通道发送当时的时间
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("客户端心跳关闭")
			return
		case <-ticker.C:
			heartbeatMsg, err := proto.Encode("/PING")
			if err != nil {
				fmt.Println("心跳 proto.Encode err:", err)
				continue
			}
			_, err = conn.Write(heartbeatMsg)
			if err != nil {
				fmt.Println("conn.Write err:", time.Now())
				fmt.Println("心跳 conn.Write err:", err)
				return
			}
		}

	}
}

// printInterface 在控制台打印漂亮的界面
func printInterface() {
	// 清屏（ANSI 转义序列）
	fmt.Print("\033[2J\033[H")
	// 打印顶部边框
	fmt.Println("\033[36m****************************************\033[0m")
	// 打印欢迎信息（蓝绿色）
	fmt.Println("\033[36m          欢迎进入聊天室                \033[0m")
	fmt.Println("\033[36m             Chat Room v1.0            \033[0m")
	fmt.Println("\033[36m****************************************\033[0m")
	fmt.Println("\033[36m***********      HELP     **************\033[0m")
	fmt.Println("\033[36m********  1.查看活跃度排行榜 /RANK *******\033[0m")
	fmt.Println("\033[36m********  2.退出聊天室      /EXIT *******\033[0m")
}
