/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-08-14
 * Description:
 * This file is part of the DataCollector project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年08月14日
 * 描述：
 --------------------------------------------------------*/

// main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)

// --- 配置常量 ---
const (
	useProxy                = true
	binanceSpotStreamURL    = "wss://stream.binance.com:9443/ws/btcusdt@trade"
	binanceFuturesStreamURL = "wss://fstream.binance.com/stream?streams=btcusdt@markPrice@1s"
	proxyURL                = "http://127.0.0.1:1080"
	natsURL                 = "nats://localhost:4222"
	natsSpotPriceSubject    = "binance.spot.price.btcusdt"
	natsFuturesPriceSubject = "binance.futures.price.btcusdt"
	natsFundingRateSubject  = "binance.futures.funding_rate.btcusdt"
)

// --- 数据结构定义 ---

// SpotTrade (币安原始数据结构)
type SpotTrade struct {
	EventType string `json:"e"`
	EventTime int64  `json:"E"`
	Symbol    string `json:"s"`
	TradeID   int64  `json:"t"`
	Price     string `json:"p"`
	Quantity  string `json:"q"`
	IsMaker   bool   `json:"m"`
}

// FuturesStreamData (币安原始数据结构)
type FuturesStreamData struct {
	Stream string          `json:"stream"`
	Data   MarkPriceUpdate `json:"data"`
}

// MarkPriceUpdate (币安原始数据结构)
type MarkPriceUpdate struct {
	EventType       string `json:"e"`
	EventTime       int64  `json:"E"`
	Symbol          string `json:"s"`
	MarkPrice       string `json:"p"`
	FundingRate     string `json:"r"`
	NextFundingTime int64  `json:"T"`
}

// ========================================================================
// [核心新增] 定义发送到NATS的统一消息格式
// 所有发送到NATS的数据都将是这个结构的JSON形式
// ========================================================================
type NatsMessage struct {
	Timestamp int64  `json:"timestamp"` // 事件发生的Unix毫秒时间戳
	Value     string `json:"value"`     // 具体的值（价格或费率）
}

// --- 主程序 (无需修改) ---
func main() {
	log.Println("--- 币安数据采集器启动 ---")
	var dialer *websocket.Dialer
	if useProxy {
		log.Printf("[INFO] 配置为使用代理。正在创建代理连接器: %s", proxyURL)
		var err error
		dialer, err = createGorillaProxyDialer(proxyURL)
		if err != nil {
			log.Fatalf("[FATAL] 创建代理拨号器失败: %v", err)
		}
	} else {
		log.Println("[INFO] 配置为不使用代理，将直接连接。")
		dialer = websocket.DefaultDialer
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("[FATAL] 连接NATS失败: %v", err)
	}
	defer nc.Close()
	log.Printf("[INFO] 已成功连接到NATS服务器: %s", natsURL)
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	var wg sync.WaitGroup
	wg.Add(2)
	go binanceWsConnectAndPublish(ctx, &wg, "现货价格", binanceSpotStreamURL, dialer, nc)
	go binanceWsConnectAndPublish(ctx, &wg, "合约价格与费率", binanceFuturesStreamURL, dialer, nc)
	<-sigChan
	log.Println("\n[INFO] 收到退出信号，正在关闭所有连接...")
	cancel()
	wg.Wait()
	log.Println("--- 币安数据采集器已安全退出 ---")
}

// createGorillaProxyDialer (无需修改)
func createGorillaProxyDialer(proxyAddr string) (*websocket.Dialer, error) {
	proxyUrl, err := url.Parse(proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("解析代理URL失败: %w", err)
	}
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyURL(proxyUrl),
		HandshakeTimeout: 45 * time.Second,
	}
	return dialer, nil
}

// binanceWsConnectAndPublish (核心逻辑修改)
func binanceWsConnectAndPublish(ctx context.Context, wg *sync.WaitGroup, name string, wsURL string, dialer *websocket.Dialer, nc *nats.Conn) {
	defer wg.Done()
	logName := fmt.Sprintf("[%s]", name)
	header := http.Header{}
	header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36")
	for {
		select {
		case <-ctx.Done():
			log.Printf("%s 收到退出信号，停止重连。", logName)
			return
		default:
			log.Printf("%s 正在连接到 %s ...", logName, wsURL)
			conn, _, err := dialer.Dial(wsURL, header)
			if err != nil {
				log.Printf("%s [ERROR] 连接失败: %v。将在5秒后重试...", logName, err)
				time.Sleep(5 * time.Second)
				continue
			}
			log.Printf("%s [SUCCESS] 连接成功！", logName)

			go func() {
				<-ctx.Done()
				log.Printf("%s 正在关闭连接...", logName)
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				conn.Close()
			}()

			for {
				messageType, msg, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
						log.Printf("%s 连接已按预期关闭。", logName)
					} else {
						log.Printf("%s [ERROR] 读取消息失败: %v", logName, err)
					}
					break
				}
				if messageType != websocket.TextMessage {
					continue
				}

				// [核心修改] 所有发布逻辑都将遵循 "创建 -> 序列化 -> 发布" 的模式
				switch name {
				case "现货价格":
					var trade SpotTrade
					if err := json.Unmarshal(msg, &trade); err != nil {
						log.Printf("%s [ERROR] 解析现货价格JSON失败: %v", logName, err)
						continue
					}
					// 创建带时间戳的消息
					natsMsg := NatsMessage{Timestamp: trade.EventTime, Value: trade.Price}
					payload, err := json.Marshal(natsMsg)
					if err != nil {
						log.Printf("%s [ERROR] 序列化现货价格消息失败: %v", logName, err)
						continue
					}
					// 发布JSON
					if err := nc.Publish(natsSpotPriceSubject, payload); err != nil {
						log.Printf("%s [ERROR] 发布现货价格到NATS失败: %v", logName, err)
					}

				case "合约价格与费率":
					var streamData FuturesStreamData
					if err := json.Unmarshal(msg, &streamData); err != nil {
						log.Printf("%s [ERROR] 解析合约流JSON失败: %v", logName, err)
						continue
					}

					// 发布合约价格 (带时间戳)
					priceMsg := NatsMessage{Timestamp: streamData.Data.EventTime, Value: streamData.Data.MarkPrice}
					pricePayload, err := json.Marshal(priceMsg)
					if err != nil {
						log.Printf("%s [ERROR] 序列化合约价格消息失败: %v", logName, err)
						continue
					}
					if err := nc.Publish(natsFuturesPriceSubject, pricePayload); err != nil {
						log.Printf("%s [ERROR] 发布合约价格到NATS失败: %v", logName, err)
					}

					// 发布资金费率 (带时间戳)
					rateMsg := NatsMessage{Timestamp: streamData.Data.EventTime, Value: streamData.Data.FundingRate}
					ratePayload, err := json.Marshal(rateMsg)
					if err != nil {
						log.Printf("%s [ERROR] 序列化资金费率消息失败: %v", logName, err)
						continue
					}
					if err := nc.Publish(natsFundingRateSubject, ratePayload); err != nil {
						log.Printf("%s [ERROR] 发布资金费率到NATS失败: %v", logName, err)
					}
				}
			}
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("%s 连接已断开，准备重连...", logName)
				time.Sleep(1 * time.Second)
			}
		}
	}
}
