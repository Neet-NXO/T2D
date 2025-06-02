package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"t2d/internal/app"
	"t2d/internal/config"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "", "配置文件路径")
	flag.Parse()

	if configFile == "" {
		fmt.Println("使用方法: t2d -config <配置文件路径>")
		fmt.Println("示例:")
		fmt.Println("  客户端: t2d -config client_config.json")
		fmt.Println("  服务端: t2d -config server_config.json")
		os.Exit(1)
	}

	// 加载配置
	cfg, err := config.Load(configFile)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 创建应用实例
	app, err := app.New(cfg)
	if err != nil {
		log.Fatalf("创建应用实例失败: %v", err)
	}

	// 启动应用
	if err := app.Start(); err != nil {
		log.Fatalf("启动应用失败: %v", err)
	}

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("正在关闭应用...")
	if err := app.Stop(); err != nil {
		log.Printf("关闭应用时出错: %v", err)
	}
	log.Println("应用已关闭")
}
