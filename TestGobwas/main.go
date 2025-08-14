/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-08-14
 * Description:
 * This file is part of the TestGobwas project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年08月14日
 * 描述：
 --------------------------------------------------------*/

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/nats-io/nats.go"
)

// --- 配置常量 ---
const (
	binanceSpotStreamURL    = "wss://stream.binance.com:9443/ws/btcusdt@trade"
	binanceFuturesStreamURL = "wss://fstream.binance.com/stream?streams=btcusdt@markPrice@1s"
	proxyURL                = "http://127.0.0.1:1080"
	natsURL                 = "nats://localhost:4222"
	natsSpotPriceSubject    = "binance.spot.price.btcusdt"
	natsFuturesPriceSubject = "binance.futures.price.btcusdt"
	natsFundingRateSubject  = "binance.futures.funding_rate.btcusdt"
)

// --- 数据结构定义 ---

type SpotTrade struct {
	EventType     string `json:"e"`
	EventTime     int64  `json:"E"`
	Symbol        string `json:"s"`
	TradeID       int64  `json:"t"`
	Price         string `json:"p"`
	Quantity      string `json:"q"`
	BuyerOrderID  int64  `json:"b"`
	SellerOrderID int64  `json:"a"`
	TradeTime     int64  `json:"T"`
}

type FuturesStreamData struct {
	Stream string          `json:"stream"`
	Data   MarkPriceUpdate `json:"data"`
}

type MarkPriceUpdate struct {
	EventType       string `json:"e"`
	EventTime       int64  `json:"E"`
	Symbol          string `json:"s"`
	MarkPrice       string `json:"p"`
	IndexPrice      string `json:"i"`
	FundingRate     string `json:"r"`
	NextFundingTime int64  `json:"T"`
}

// --- 主程序 ---
func main() {
	log.Println("--- 币安数据采集器启动 ---")

	dialer, err := createProxyDialer(proxyURL)
	if err != nil {
		log.Fatalf("[FATAL] 创建代理拨号器失败: %v", err)
	}
	log.Printf("[INFO] 已配置HTTP代理: %s", proxyURL)

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

// createProxyDialer 创建一个支持HTTP代理且伪装成浏览器的ws.Dialer
func createProxyDialer(proxyAddr string) (ws.Dialer, error) {
	proxyUrl, err := url.Parse(proxyAddr)
	if err != nil {
		return ws.Dialer{}, fmt.Errorf("解析代理URL失败: %w", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyUrl),
	}

	// ========================================================================
	// [核心修改] 创建一个HTTP Header，并添加标准的User-Agent
	// 这会让我们的连接请求看起来像一个普通的浏览器，从而绕过服务器的检测
	// ========================================================================
	header := http.Header{}
	header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36")

	return ws.Dialer{
		NetDial: transport.DialContext,
		Header:  ws.HandshakeHeaderHTTP(header), // <--- 将我们伪造的Header应用到拨号器
	}, nil
}

// binanceWsConnectAndPublish 是一个通用的WebSocket处理函数
func binanceWsConnectAndPublish(ctx context.Context, wg *sync.WaitGroup, name string, wsURL string, dialer ws.Dialer, nc *nats.Conn) {
	defer wg.Done()
	logName := fmt.Sprintf("[%s]", name)

	for {
		select {
		case <-ctx.Done():
			log.Printf("%s 收到退出信号，停止重连。", logName)
			return
		default:
			log.Printf("%s 正在连接到 %s ...", logName, wsURL)
			conn, _, _, err := dialer.Dial(ctx, wsURL)
			if err != nil {
				log.Printf("%s [ERROR] 连接失败: %v。将在5秒后重试...", logName, err)
				time.Sleep(5 * time.Second)
				continue
			}
			log.Printf("%s [SUCCESS] 连接成功！", logName)

			for {
				msg, op, err := wsutil.ReadServerData(conn)
				if err != nil {
					var closeErr wsutil.ClosedError
					if errors.As(err, &closeErr) {
						log.Printf("%s [WARN] 连接已正常关闭: %v", logName, closeErr)
					} else {
						log.Printf("%s [ERROR] 读取消息失败: %v", logName, err)
					}
					conn.Close()
					break
				}

				if op != ws.OpText {
					continue
				}

				switch name {
				case "现货价格":
					var trade SpotTrade
					if err := json.Unmarshal(msg, &trade); err != nil {
						log.Printf("%s [ERROR] 解析现货价格JSON失败: %v. Raw: %s", logName, err, string(msg))
						continue
					}
					if err := nc.Publish(natsSpotPriceSubject, []byte(trade.Price)); err != nil {
						log.Printf("%s [ERROR] 发布现货价格到NATS失败: %v", logName, err)
					}

				case "合约价格与费率":
					var streamData FuturesStreamData
					if err := json.Unmarshal(msg, &streamData); err != nil {
						log.Printf("%s [ERROR] 解析合约流JSON失败: %v. Raw: %s", logName, err, string(msg))
						continue
					}
					if err := nc.Publish(natsFuturesPriceSubject, []byte(streamData.Data.MarkPrice)); err != nil {
						log.Printf("%s [ERROR] 发布合约价格到NATS失败: %v", logName, err)
					}
					if err := nc.Publish(natsFundingRateSubject, []byte(streamData.Data.FundingRate)); err != nil {
						log.Printf("%s [ERROR] 发布资金费率到NATS失败: %v", logName, err)
					}
				}
			}
			log.Printf("%s 连接已断开，准备重连...", logName)
		}
	}
}
