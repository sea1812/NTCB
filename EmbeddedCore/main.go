/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-08-13
 * Description:
 * This file is part of the EmbeddedCore project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年08月13日
 * 描述：嵌入式的NATS Core服务器
 --------------------------------------------------------*/

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func main() {
	// 1. 配置嵌入式NATS服务器
	opts := &server.Options{
		Host:   "localhost", // 绑定地址
		Port:   4222,        // 端口
		NoLog:  false,       // 开启日志（开发环境便于调试）
		NoSigs: true,        // 禁用服务器自带的信号处理（由主程序统一处理）
	}

	// 2. 启动嵌入式NATS服务器
	s, err := server.NewServer(opts)
	if err != nil {
		log.Fatalf("启动NATS服务器失败: %v", err)
	}

	// 非阻塞启动服务器
	go s.Start()

	// 等待服务器就绪
	if !s.ReadyForConnections(5 * time.Second) {
		log.Fatal("NATS服务器未能在指定时间内启动")
	}
	fmt.Println("✅ 嵌入式NATS服务器已启动，地址: nats://localhost:4222")
	fmt.Println("📝 服务器运行中，按 Ctrl+C 退出...")

	// 3. 连接到嵌入式NATS服务器
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatalf("连接NATS服务器失败: %v", err)
	}
	defer nc.Close() // 退出前关闭客户端连接

	// 4. 启动订阅者（持续接收消息）
	subscriberCount := 2
	for i := 1; i <= subscriberCount; i++ {
		subID := i
		_, err := nc.Subscribe("demo.topic", func(msg *nats.Msg) {
			fmt.Printf("[订阅者%d] 收到消息: %s (主题: %s)\n", subID, string(msg.Data), msg.Subject)
		})
		if err != nil {
			log.Printf("订阅者%d 订阅失败: %v", subID, err)
		} else {
			fmt.Printf("✅ 订阅者%d 已订阅主题: demo.topic\n", subID)
		}
	}

	// 5. 启动发布者（发送一批测试消息后保持运行）
	go func() {
		// 发送3条测试消息
		for i := 1; i <= 3; i++ {
			msg := fmt.Sprintf("测试消息 %d (时间: %s)", i, time.Now().Format("15:04:05"))
			if err := nc.Publish("demo.topic", []byte(msg)); err != nil {
				log.Printf("发布消息失败: %v", err)
			} else {
				fmt.Printf("[发布者] 发送消息: %s\n", msg)
			}
			time.Sleep(1 * time.Second)
		}
		// 消息发送完成后不退出，保持程序运行
		fmt.Println("📤 测试消息发送完成，服务器持续运行中...")
	}()

	// 6. 等待系统中断信号（Ctrl+C 或 kill 命令）
	sigChan := make(chan os.Signal, 1)
	// 监听 SIGINT (Ctrl+C) 和 SIGTERM (kill 命令)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞等待信号
	<-sigChan
	fmt.Println("\n📢 收到退出信号，开始优雅关闭...")

	// 7. 优雅关闭资源
	nc.Drain() // 先关闭客户端连接（确保消息处理完成）
	fmt.Println("🔌 客户端连接已关闭")

	s.Shutdown()                      // 关闭NATS服务器
	fmt.Println("🛑 NATS服务器已关闭") // 移除ReadyForShutdown判断

	fmt.Println("👋 程序已退出")
}
