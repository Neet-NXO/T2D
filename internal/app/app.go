package app

import (
	"fmt"

	"t2d/internal/client"
	"t2d/internal/config"
	"t2d/internal/server"
)

// App 应用程序接口
type App interface {
	Start() error
	Stop() error
	Run() error
}

// New 根据配置创建应用实例
func New(cfg *config.Config) (App, error) {
	if cfg.IsClient() {
		return client.New(cfg)
	} else if cfg.IsServer() {
		return server.New(cfg)
	} else {
		return nil, fmt.Errorf("invalid mode: %s, must be 'client' or 'server'", cfg.Mode)
	}
}
