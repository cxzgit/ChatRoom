package utils

import (
	"fmt"
	"gopkg.in/ini.v1"
)

// Config 结构体存放配置信息
type Config struct {
	ServerAddress     string
	ServerPort        string
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
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
		ServerAddress:     cfg.Section("server").Key("ServerAddress").MustString("127.0.0.1"),
		ServerPort:        cfg.Section("server").Key("port").MustString("8888"),
		RedisAddr:         cfg.Section("redis").Key("addr").MustString("192.168.24.101:6379"),
		RedisPassword:     cfg.Section("redis").Key("password").String(),
		RedisDB:           cfg.Section("redis").Key("db").MustInt(0),
		HeartbeatInterval: cfg.Section("heartbeat").Key("clientInterval").MustInt(5),
		LogFile:           cfg.Section("log").Key("file").MustString("server.log"),
	}
	fmt.Println("配置加载成功")
	return config, nil
}
