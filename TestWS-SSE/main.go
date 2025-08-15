/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-08-15
 * Description:
 * This file is part of the TestWS-SSE project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年08月15日
 * 描述：测试WebSocket推送事件
 --------------------------------------------------------*/

// main.go
package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"
)

// 统一事件结构（前后端约定）
type Event struct {
	Type string      `json:"type"` // 事件类型标识
	Data interface{} `json:"data"` // 事件具体数据
}

// 系统状态数据结构
type SystemStatus struct {
	Status string `json:"status"`
	Load   int    `json:"load"`
}

// 通知数据结构
type Notification struct {
	Message  string `json:"message"`
	Priority string `json:"priority"`
}

// 进度数据结构
type Progress struct {
	Task    string `json:"task"`
	Percent int    `json:"percent"`
}

// 警告数据结构
type Warning struct {
	Message string `json:"message"`
}

// 信息数据结构
type Info struct {
	Message string `json:"message"`
}

func main() {
	s := g.Server()

	// 静态文件服务（前端页面）
	s.AddStaticPath("/", "./public")

	// WebSocket连接处理端点
	mGroup := s.Group("/")
	mGroup.GET("/ws", func(r *ghttp.Request) {
		// 升级HTTP连接为WebSocket
		ws, err := r.WebSocket()
		if err != nil {
			g.Log().Error(r.Context(), "WebSocket升级失败:", err)
			r.Response.WriteStatusExit(500, "WebSocket升级失败")
			return
		}
		defer func() {
			ws.Close()
			g.Log().Info(r.Context(), "客户端WebSocket连接已关闭")
		}()

		g.Log().Info(r.Context(), "客户端已建立WebSocket连接")

		// 发送连接成功确认事件
		sendEvent(r.Context(), ws, "connect", map[string]string{"status": "connected"})

		// 启动独立goroutine处理客户端消息（保持连接活跃，检测断开）
		done := make(chan struct{})
		go func() {
			for {
				// 读取客户端消息（无实际业务逻辑，仅用于检测断开）
				_, _, err := ws.ReadMessage()
				if err != nil {
					g.Log().Info(r.Context(), "客户端断开连接:", err)
					close(done)
					return
				}
			}
		}()

		// 事件发送循环
		counter := 0
		for {
			select {
			case <-done: // 客户端断开连接
				return
			default:
				counter++
				time.Sleep(1 * time.Second) // 每秒发送一个事件

				// 轮询发送不同类型事件
				switch counter % 4 {
				case 0:
					// 系统状态事件
					data := SystemStatus{
						Status: "正常",
						Load:   counter % 100, // 0-99的负载值
					}
					sendEvent(r.Context(), ws, "system-status", data)

				case 1:
					// 通知事件
					data := Notification{
						Message:  "新消息 #" + gconv.String(counter),
						Priority: "normal",
					}
					sendEvent(r.Context(), ws, "notification", data)

				case 2:
					// 进度事件
					progress := (counter % 10) * 10 // 0-90的进度值
					data := Progress{
						Task:    "数据同步",
						Percent: progress,
					}
					sendEvent(r.Context(), ws, "progress", data)

				case 3:
					// 警告或信息事件（每20次发送一次警告）
					if counter%20 == 0 {
						data := Warning{
							Message: "资源使用率过高 (" + gconv.String(70+counter%30) + "%)",
						}
						sendEvent(r.Context(), ws, "warning", data)
					} else {
						data := Info{
							Message: "系统运行正常，当前时间: " + time.Now().Format("15:04:05"),
						}
						sendEvent(r.Context(), ws, "info", data)
					}
				}
			}
		}
	})

	// 启动服务器
	s.SetPort(8081)
	g.Log().Info(context.Background(), "服务器启动于: http://localhost:8080")
	s.Run()
}

// 发送事件到WebSocket客户端
func sendEvent(ctx context.Context, ws *ghttp.WebSocket, eventType string, data interface{}) error {
	// 构建事件
	event := Event{
		Type: eventType,
		Data: data,
	}

	// 序列化事件为JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		g.Log().Error(ctx, "事件JSON序列化失败:", err)
		return err
	}

	// 发送文本消息
	if err := ws.WriteMessage(ghttp.WsMsgText, jsonData); err != nil {
		g.Log().Error(ctx, "WebSocket消息发送失败:", err)
		return err
	}

	g.Log().Debug(ctx, "发送事件成功:", eventType)
	return nil
}
