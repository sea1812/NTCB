/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-08-09
 * Description:
 * This file is part of the TestTSDB project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年08月09日
 * 描述：测试Prometheus TSDB、币安WebSocket行情读取和协程读写
 --------------------------------------------------------*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/prometheus/prometheus/tsdb/chunkenc"
	"golang.org/x/net/proxy"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	dbPath = "./binance_tsdb_data"

	// 币安 WebSocket API 地址
	spotStreamURL    = "wss://stream.binance.com:9443/ws/btcusdt@trade"
	futuresStreamURL = "wss://fstream.binance.com/ws/btcusdt@aggTrade"

	// TSDB 中的指标名称
	spotMetricName    = "binance_spot_price"
	futuresMetricName = "binance_futures_price"
)

// 增加 EventTime 字段，对应 JSON 中的 "E" 使用 interface{} 来接收可能是 string 或 number 的字段
type BinanceStreamPayload struct {
	Stream       string      `json:"s"` // Symbol
	Price        string      `json:"p"` // Price
	EventTimeRaw interface{} `json:"E"` // Event time (raw, can be string or number)
}

func main() {
	// 1. 设置 TSDB
	os.RemoveAll(dbPath)
	db, err := tsdb.Open(dbPath, nil, nil, tsdb.DefaultOptions(), nil)
	if err != nil {
		log.Fatalf("FATAL: 打开 TSDB 失败: %v", err)
	}
	defer db.Close()

	// 2. 设置优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	// --- 协程 A: 启动写入器 ---
	wg.Add(1)
	go writerRoutine(ctx, &wg, db)

	// --- 协程 B: 启动读取器 ---
	wg.Add(1)
	go readerRoutine(ctx, &wg, db)

	log.Println("应用已启动，按 Ctrl+C 退出。")
	<-termChan // 等待中断信号
	log.Println("收到关闭信号，正在优雅退出...")
	cancel()  // 通知所有协程退出
	wg.Wait() // 等待所有协程完成
	log.Println("所有协程已安全退出。")
}

// writerRoutine (协程A) 负责管理WebSocket连接并写入数据
func writerRoutine(ctx context.Context, wg *sync.WaitGroup, db *tsdb.DB) {
	defer wg.Done()
	var writerWg sync.WaitGroup

	// 为现货和合约分别启动一个连接和读取的子协程
	writerWg.Add(2)
	go connectAndWrite(ctx, &writerWg, db, spotStreamURL, spotMetricName)
	go connectAndWrite(ctx, &writerWg, db, futuresStreamURL, futuresMetricName)

	writerWg.Wait()
	log.Println("[Writer] 写入协程已停止。")
}

// connectAndWrite 完整的最终版本，包含了代理和完整的读写循环
/*
func connectAndWrite(ctx context.Context, wg *sync.WaitGroup, db *tsdb.DB, urlString, metricName string) {
	defer wg.Done()
	logName := fmt.Sprintf("[%s]", metricName)

	// --- 代理配置 (最终修正) ---
	// 使用 socks5h 来让代理服务器执行 DNS 解析
	//proxyAddress := "socks5h://127.0.0.1:1080" // <--- 关键修改！
	proxyAddress := "http://127.0.0.1:1080" // <--- 使用http代理
	//也可以使用HTTP代理""

	var dialer *websocket.Dialer
	if proxyAddress != "" {
		proxyURL, err := url.Parse(proxyAddress)
		if err != nil {
			log.Fatalf("FATAL: 解析代理URL失败: %v", err)
		}
		proxyDialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			log.Fatalf("FATAL: 创建代理拨号器失败: %v", err)
		}
		dialer = &websocket.Dialer{
			NetDial:          proxyDialer.Dial,
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 45 * time.Second,
		}
		log.Printf("%s 将通过代理 %s (使用远程解析) 进行连接", logName, proxyURL.Host)
	} else {
		dialer = websocket.DefaultDialer
		log.Printf("%s 将进行直接连接", logName)
	}
	// --- 代理配置结束 ---

	// ... 后续代码保持不变 ...
	for {
		select {
		case <-ctx.Done():
			log.Printf("%s 收到退出信号，停止连接。", logName)
			return
		default:
			conn, _, err := dialer.DialContext(ctx, urlString, nil)
			if err != nil {
				log.Printf("%s 连接失败: %v. 5秒后重试...", logName, err)
				time.Sleep(5 * time.Second)
				continue
			}
			log.Printf("%s 连接成功！", logName)

		readLoop:
			for {
				if ctx.Err() != nil {
					log.Printf("%s 上下文已取消，关闭连接。", logName)
					conn.Close()
					return
				}
				_, message, err := conn.ReadMessage()
				if err != nil {
					log.Printf("%s 读取消息失败: %v. 准备重连...", logName, err)
					conn.Close()
					break readLoop
				}

				var payload BinanceStreamPayload
				if err := json.Unmarshal(message, &payload); err != nil {
					log.Printf("%s [ERROR] JSON解析失败: %v", logName, err)
					continue
				}
				price, err := strconv.ParseFloat(payload.Price, 64)
				if err != nil {
					log.Printf("%s [ERROR] 价格转换失败: %v", logName, err)
					continue
				}
				app := db.Appender(ctx)
				_, err = app.Append(0, labels.FromStrings("__name__", metricName), time.Now().UnixMilli(), price)
				if err != nil {
					log.Printf("%s [ERROR] TSDB Append 失败: %v", logName, err)
					app.Rollback()
					continue
				}
				if err := app.Commit(); err != nil {
					log.Printf("%s [ERROR] TSDB Commit 失败: %v", logName, err)
					continue
				}
				log.Printf("%s [SUCCESS] 成功写入价格: %.2f", logName, price)
			}
			time.Sleep(1 * time.Second)
		}
	}
}

*/
// connectAndWrite 完整的最终版本，同时支持 SOCKS5 和 HTTP 代理
/*
func connectAndWrite(ctx context.Context, wg *sync.WaitGroup, db *tsdb.DB, urlString, metricName string) {
	defer wg.Done()
	logName := fmt.Sprintf("[%s]", metricName)

	// --- 代理配置 (支持 http 和 socks5h) ---
	// 在这里填入代理地址，可以是 http 或 socks5h
	proxyAddress := "http://127.0.0.1:1080"
	// proxyAddress := "socks5h://127.0.0.1:1080" // 或者使用 SOCKS5h

	dialer := websocket.DefaultDialer // 创建一个默认的拨号器

	if proxyAddress != "" {
		proxyURL, err := url.Parse(proxyAddress)
		if err != nil {
			log.Fatalf("FATAL: 解析代理URL失败: %v", err)
		}

		// 根据代理的协议类型，使用不同的配置方法
		switch proxyURL.Scheme {
		case "socks5", "socks5h":
			// --- SOCKS 代理的配置方法 ---
			log.Printf("%s 检测到 SOCKS 代理，将通过 %s (使用远程解析) 进行连接", logName, proxyURL.Host)
			proxyDialer, err := proxy.FromURL(proxyURL, proxy.Direct)
			if err != nil {
				log.Fatalf("FATAL: 创建 SOCKS 代理拨号器失败: %v", err)
			}
			// 修改网络层的拨号函数
			dialer.NetDial = proxyDialer.Dial

		case "http", "https":
			// --- HTTP 代理的配置方法 ---
			log.Printf("%s 检测到 HTTP 代理，将通过 %s 进行连接", logName, proxyURL.Host)
			// 设置应用层的代理函数
			dialer.Proxy = http.ProxyURL(proxyURL)

		default:
			log.Fatalf("FATAL: 不支持的代理协议: %s", proxyURL.Scheme)
		}
	} else {
		log.Printf("%s 将进行直接连接", logName)
	}
	// --- 代理配置结束 ---

	// ... 后续的连接和读写循环保持完全不变 ...
	for {
		select {
		case <-ctx.Done():
			log.Printf("%s 收到退出信号，停止连接。", logName)
			return
		default:
			conn, _, err := dialer.DialContext(ctx, urlString, nil)
			if err != nil {
				log.Printf("%s 连接失败: %v. 5秒后重试...", logName, err)
				time.Sleep(5 * time.Second)
				continue
			}
			log.Printf("%s 连接成功！", logName)

		readLoop:
			for {
				if ctx.Err() != nil {
					log.Printf("%s 上下文已取消，关闭连接。", logName)
					conn.Close()
					return
				}
				_, message, err := conn.ReadMessage()
				if err != nil {
					log.Printf("%s 读取消息失败: %v. 准备重连...", logName, err)
					conn.Close()
					break readLoop
				}

				var payload BinanceStreamPayload
				if err := json.Unmarshal(message, &payload); err != nil {
					log.Printf("%s [ERROR] JSON解析失败: %v", logName, err)
					continue
				}
				price, err := strconv.ParseFloat(payload.Price, 64)
				if err != nil {
					log.Printf("%s [ERROR] 价格转换失败: %v", logName, err)
					continue
				}
				app := db.Appender(ctx)
				//_, err = app.Append(0, labels.FromStrings("__name__", metricName), time.Now().UnixMilli(), price)
				// 使用 payload.EventTime 作为时间戳
				_, err = app.Append(0, labels.FromStrings("__name__", metricName), payload.EventTime, price)
				if err != nil {
					// 这里的错误处理逻辑可能需要微调，见下文解释
					if strings.Contains(err.Error(), "out of order") {
						// 如果是乱序错误，可以选择性忽略并继续
						log.Printf("%s [WARN] 收到乱序样本，已忽略: %v", logName, err)
						app.Rollback() // 必须回滚失败的事务
						continue
					}
					log.Printf("%s [ERROR] TSDB Append 失败: %v", logName, err)
					app.Rollback()
					continue
				}
				if err != nil {
					log.Printf("%s [ERROR] TSDB Append 失败: %v", logName, err)
					app.Rollback()
					continue
				}
				if err := app.Commit(); err != nil {
					log.Printf("%s [ERROR] TSDB Commit 失败: %v", logName, err)
					continue
				}
				log.Printf("%s [SUCCESS] 成功写入价格: %.2f", logName, price)
			}
			time.Sleep(1 * time.Second)
		}
	}
}
*/

// connectAndWrite 最终、完整、健壮的版本，支持优雅退出
func connectAndWrite(ctx context.Context, wg *sync.WaitGroup, db *tsdb.DB, urlString, metricName string) {
	defer wg.Done()
	logName := fmt.Sprintf("[%s]", metricName)

	// --- 代理配置部分保持不变 ---
	proxyAddress := "http://127.0.0.1:1080" // 使用你成功的代理配置
	dialer := websocket.DefaultDialer
	if proxyAddress != "" {
		proxyURL, err := url.Parse(proxyAddress)
		if err != nil {
			log.Fatalf("FATAL: 解析代理URL失败: %v", err)
		}
		switch proxyURL.Scheme {
		case "http", "https":
			dialer.Proxy = http.ProxyURL(proxyURL)
		case "socks5", "socks5h":
			proxyDialer, err := proxy.FromURL(proxyURL, proxy.Direct)
			if err != nil {
				log.Fatalf("FATAL: 创建 SOCKS 代理拨号器失败: %v", err)
			}
			dialer.NetDial = proxyDialer.Dial
		default:
			log.Fatalf("FATAL: 不支持的代理协议: %s", proxyURL.Scheme)
		}
		log.Printf("%s 将通过代理 %s 进行连接", logName, proxyURL.Scheme)
	} else {
		log.Printf("%s 将进行直接连接", logName)
	}
	// --- 代理配置结束 ---

	// 外层循环负责断线重连
	for {
		// 在每次尝试重连前，检查是否收到了退出信号
		select {
		case <-ctx.Done():
			log.Printf("%s 收到退出信号，停止重连循环。", logName)
			return
		default:
			// 继续执行连接
		}

		conn, _, err := dialer.DialContext(ctx, urlString, nil)
		if err != nil {
			log.Printf("%s 连接失败: %v. 5秒后重试...", logName, err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Printf("%s 连接成功！", logName)

		// --- 优雅退出的核心逻辑 ---
		// 启动一个专用的 "closer" 协程，它只负责监听退出信号并关闭连接
		go func() {
			<-ctx.Done()
			log.Printf("%s [Closer] 收到退出信号，正在关闭连接...", logName)
			conn.Close()
		}()
		// --- 核心逻辑结束 ---

		// 内层循环，只负责读取当前连接的消息
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				// 当连接被关闭时，这里会收到一个错误
				// 我们需要判断这是不是一个预期的关闭
				select {
				case <-ctx.Done():
					// 如果上下文已取消，说明是正常的优雅退出
					log.Printf("%s 连接已按预期关闭。", logName)
				default:
					// 如果上下文没有取消，说明是网络错误或对方关闭了连接
					log.Printf("%s 读取消息失败（准备重连）: %v", logName, err)
				}
				// 无论哪种情况，都跳出内层循环，外层循环会决定是否重连
				break
			}

			// --- 数据处理部分保持不变 ---
			var payload BinanceStreamPayload
			if err := json.Unmarshal(message, &payload); err != nil {
				log.Printf("%s [ERROR] JSON解析失败: %v", logName, err)
				continue
			}
			price, err := strconv.ParseFloat(payload.Price, 64)
			if err != nil {
				log.Printf("%s [ERROR] 价格转换失败: %v", logName, err)
				continue
			}
			var eventTime int64
			switch v := payload.EventTimeRaw.(type) {
			case float64:
				eventTime = int64(v)
			case string:
				parsedTime, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					log.Printf("%s [ERROR] 事件时间字符串解析失败: %v", logName, err)
					continue
				}
				eventTime = parsedTime
			default:
				log.Printf("%s [ERROR] 未知的事件时间类型: %T", logName, v)
				continue
			}
			app := db.Appender(ctx)
			_, err = app.Append(0, labels.FromStrings("__name__", metricName), eventTime, price)
			if err != nil {
				if strings.Contains(err.Error(), "out of order") || strings.Contains(err.Error(), "duplicate sample for timestamp") {
					app.Rollback()
					continue
				}
				log.Printf("%s [ERROR] TSDB Append 失败: %v", logName, err)
				app.Rollback()
				continue
			}
			if err := app.Commit(); err != nil {
				log.Printf("%s [ERROR] TSDB Commit 失败: %v", logName, err)
				continue
			}
			//log.Printf("%s [SUCCESS] 成功写入价格: %.2f", logName, price)
		}
	}
}

// readerRoutine (协程B) 负责读取最新数据并计算差额
func readerRoutine(ctx context.Context, wg *sync.WaitGroup, db *tsdb.DB) {
	defer wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Println("[Reader] 读取和计算协程已启动。")
	for {
		select {
		case <-ctx.Done():
			log.Println("[Reader] 收到退出信号，停止读取。")
			return
		case <-ticker.C:
			spotPrice, spotFound := getLatestValue(db, spotMetricName)
			futuresPrice, futuresFound := getLatestValue(db, futuresMetricName)

			if spotFound && futuresFound {
				spread := futuresPrice - spotPrice
				log.Printf("--- 最新价格 | 现货: %.2f | 合约: %.2f | 差额 (合约-现货): %.2f ---\n", spotPrice, futuresPrice, spread)
			} else {
				// 使用 Printf 来正确格式化字符串
				log.Printf("--- 等待数据中... (现货找到: %v, 合约找到: %v) ---\n", spotFound, futuresFound)
			}
		}
	}
}

// getLatestValue 从TSDB查询指定指标的最新值 (根据最新报错信息最终修正版)
func getLatestValue(db *tsdb.DB, metricName string) (float64, bool) {
	// 查询过去1分钟的数据，确保能找到最新的点
	minTime := time.Now().Add(-1 * time.Minute).UnixMilli()
	maxTime := time.Now().UnixMilli()

	// 1. Querier 调用 (根据之前的报错，此版本不带 context)
	q, err := db.Querier(minTime, maxTime)
	if err != nil {
		log.Printf("[Query] 创建查询器失败: %v", err)
		return 0, false
	}
	defer q.Close()

	// 创建标签匹配器
	matcher, err := labels.NewMatcher(labels.MatchEqual, "__name__", metricName)
	if err != nil {
		log.Printf("[Query] 创建匹配器失败: %v", err)
		return 0, false
	}

	// 2. Select 调用 (根据最新的报错，使用其期望的完整签名)
	// 签名: Select(ctx context.Context, sortSeries bool, hints *storage.SelectHints, matchers ...*labels.Matcher)
	// - ctx: 我们传入一个背景上下文 context.Background()
	// - sortSeries: 我们不需要对序列排序，传入 false
	// - hints: 我们没有查询提示，传入 nil
	// - matchers: 传入我们创建的 matcher
	ss := q.Select(context.Background(), false, nil, matcher)

	var lastVal float64
	var found bool

	// 遍历查询结果集
	for ss.Next() {
		series := ss.At()
		// 迭代序列中的所有数据点以找到最后一个点
		it := series.Iterator(nil)
		for it.Next() != chunkenc.ValNone {
			_, val := it.At()
			lastVal = val
			found = true
		}
	}

	if ss.Err() != nil {
		log.Printf("[Query] 查询出错: %v", ss.Err())
		return 0, false
	}

	return lastVal, found
}
