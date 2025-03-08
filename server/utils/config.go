package utils

import (
	"fmt"
	"gopkg.in/ini.v1"
)

// Config 结构体存放配置信息
type Config struct {
	ServerPort        string
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	HeartbeatTimeout  int
	HeartbeatInterval int
	LogFile           string
}

// LoadConfig 从 config.ini 文件加载配置信息
func LoadConfig() (*Config, error) {
	cfg, err := ini.Load("gocode/RedisExam/chatRoom/config.ini")
	if err != nil {
		fmt.Println("加载.ini时出错")
		return nil, err
	}

	config := &Config{
		ServerPort:        cfg.Section("server").Key("port").MustString("8888"),
		RedisAddr:         cfg.Section("redis").Key("addr").MustString("192.168.24.101:6379"),
		RedisPassword:     cfg.Section("redis").Key("password").String(),
		RedisDB:           cfg.Section("redis").Key("db").MustInt(0),
		HeartbeatTimeout:  cfg.Section("heartbeat").Key("timeout").MustInt(30),
		HeartbeatInterval: cfg.Section("heartbeat").Key("interval").MustInt(10),
		LogFile:           cfg.Section("log").Key("file").MustString("server.log"),
	}

	fmt.Println("配置加载成功")
	return config, nil
}
