package config

import (
	"github.com/spf13/viper"

	"GoMusic/initialize/log"
)

const (
	DebugEnv = "DEBUG"
)

var AllConfig Server

type Server struct {
	Port  int `mapstructure:"port"`
	Redis struct {
		Dsn      string `mapstructure:"dsn"`
		Password string `mapstructure:"password"`
	}
}

func init() {
	v := viper.New()
	configPath := "./config.yaml"
	v.SetConfigFile(configPath)
	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		log.Errorf("read config file error: %v", err)
	}
	if err := v.Unmarshal(&AllConfig); err != nil {
		log.Errorf("unmarshal config file error: %v", err)
	}
	log.Info("config load success...")
}
