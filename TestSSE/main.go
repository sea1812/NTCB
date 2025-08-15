/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-08-15
 * Description:
 * This file is part of the TestSSE project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年08月15日
 * 描述：测试单端点多事件的SSE机制
 --------------------------------------------------------*/

package main

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"
	"time"
)

func main() {
	s := g.Server()

	// 提供静态文件服务，用于前端页面
	s.AddStaticPath("/", "./static")

	// SSE端点，处理客户端连接
	mGroup := s.Group("/")
	mGroup.GET("/events", func(r *ghttp.Request) {
		// 设置SSE响应头
		r.Response.Header().Set("Content-Type", "text/event-stream")
		r.Response.Header().Set("Cache-Control", "no-cache")
		r.Response.Header().Set("Connection", "keep-alive")
		r.Response.Header().Set("Access-Control-Allow-Origin", "*")

		// 用于控制循环退出
		done := make(chan struct{})
		defer close(done)

		// 监听客户端断开连接
		go func() {
			<-r.Context().Done()
			done <- struct{}{}
		}()

		// 计数器，用于演示
		counter := 0

		// 循环发送不同类型的事件
		for {
			select {
			case <-done:
				return
			default:
				counter++

				// 每隔1秒发送一个事件，轮流发送不同类型
				switch counter % 4 {
				case 0:
					// 系统状态事件
					r.Response.WriteString("event: system-status\n")
					r.Response.WriteString("data: {\"status\": \"正常\", \"load\": " + gconv.String(counter%100) + "}\n\n")

				case 1:
					// 通知事件
					r.Response.WriteString("event: notification\n")
					r.Response.WriteString("data: {\"message\": \"新消息 #" + gconv.String(counter) + "\", \"priority\": \"normal\"}\n\n")

				case 2:
					// 进度事件
					progress := (counter % 10) * 10
					r.Response.WriteString("event: progress\n")
					r.Response.WriteString("data: {\"task\": \"数据同步\", \"percent\": " + gconv.String(progress) + "}\n\n")

				case 3:
					// 警告事件
					if counter%20 == 0 { // 偶尔发送警告
						r.Response.WriteString("event: warning\n")
						r.Response.WriteString("data: {\"message\": \"资源使用率过高 (" + gconv.String(70+counter%30) + "%)\"}\n\n")
					} else {
						// 信息事件
						r.Response.WriteString("event: info\n")
						r.Response.WriteString("data: {\"message\": \"系统运行正常，当前时间: " + time.Now().Format("15:04:05") + "\"}\n\n")
					}
				}

				// 刷新缓冲区，确保数据被发送
				r.Response.Flush()

				// 等待1秒
				time.Sleep(1 * time.Second)
			}
		}
	})

	// 启动服务器
	s.SetPort(8080)
	s.Run()
}
