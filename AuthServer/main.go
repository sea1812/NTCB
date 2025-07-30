/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-07-29
 * Description:
 * This file is part of the AuthServer project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年07月29日
 * 描述：NTCB工程中Auth Server，处理新程序注册事务
 --------------------------------------------------------*/

package main

import (
	NTPack "AuthServer/App"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"time"
)

var (
	// 声明全局变量
	AccessKey    string
	PublisherId  string
	ComponentId  string
	StartTime    time.Time
	ServerNodeId int64

	// 全局对象 MQTT客户端
	MqttClient mqtt.Client
)

func main() {
	//填充全局变量
	mCtx := gctx.New()
	mAccessKey, _ := g.Config().Get(mCtx, "ntcb.accessKey")
	mPublisherId, _ := g.Config().Get(mCtx, "ntcb.publisherId")
	mComponentId, _ := g.Config().Get(mCtx, "ntcb.componentId")
	mServerNodeId, _ := g.Config().Get(mCtx, "ntcb.serverNodeId")
	AccessKey = mAccessKey.String()
	PublisherId = mPublisherId.String()
	ComponentId = mComponentId.String()
	ServerNodeId = mServerNodeId.Int64()
	StartTime = time.Now()
	//生成ComponentHeader数据
	mHeader := NTPack.NewComponentHeader(ServerNodeId)
	//创建MQTT客户端
	InitMqttClient(*mHeader)
	fmt.Println("Testing message server...")
	//TODO 检查消息服务器是否可用，如果不可用则退出进程
	if token := MqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer MqttClient.Disconnect(250) // 优雅断开连接，等待250ms处理剩余消息

	//设置Http路径
	s := g.Server()
	mGroup := s.Group("/")
	mGroup.GET("/about", GetAbout) //返回版本信息
	mGroup.GET("/list", GetList)   //返回已注册程序列表
	mGroup.POST("/reg", PostReg)   //注册新设备

	//清理Components表
	_, _ = g.DB().Model("Components").Delete("1=1")
	//将自身Header数据插入Components表
	_, _ = g.DB().Model("Components").Insert(gjson.New(mHeader))

	//TODO 广播Public/Enter消息Auth服务上线
	mHeader.AccessKey = "hidden"
	MqttClient.Publish("ntcb/enter", 0, false, gjson.New(mHeader).String())
	//TODO 同时发布到日志频道

	s.Run()
}

func MqttOnConnect(client mqtt.Client) {
	fmt.Println("Connected")
	//订阅频道
	MqttClient.Subscribe("ntcb/#", 0, nil)
}

func MqttOnLostConnect(client mqtt.Client, err error) {
	fmt.Println("LostConnect")
}

func MqttOnMessage(client mqtt.Client, Message mqtt.Message) {
	fmt.Println("OnMessage___________________")
	fmt.Println(Message.Topic())
	fmt.Println(string(Message.Payload()))
}

// InitMqttClient 初始化Mqtt客户端
func InitMqttClient(AComponent NTPack.TCBComponentHeader) {
	//从Config文件读取参数
	mCtx := gctx.New()
	mReconnectDuration, _ := g.Config().Get(mCtx, "ntcb.reconnectDuration") //最大重新连接时间
	mBroker, _ := g.Config().Get(mCtx, "ntcb.broker")                       //MQTT服务器地址
	mBrokerUser, _ := g.Config().Get(mCtx, "ntcb.brokerUser")               //MQTT服务器用户
	mBrokerPassword, _ := g.Config().Get(mCtx, "ntcb.brokerPassword")       //MQTT服务器用户密码

	opts := mqtt.NewClientOptions()
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(30 * time.Second) //设置超时重试时间为15秒
	opts.SetKeepAlive(15 * time.Second)            //设置心跳为15秒
	//客户端ID为Component，本地IP和进程PID的组合
	opts.SetClientID(fmt.Sprintf("%s_%s_%d", AComponent.ComponentID, AComponent.LocalIP, AComponent.Pid))
	opts.SetMaxReconnectInterval(time.Duration(mReconnectDuration.Int()) * time.Second)
	opts.AddBroker(mBroker.String())
	//设置Mqtt客户端事件
	opts.SetOnConnectHandler(MqttOnConnect)
	opts.SetConnectionLostHandler(MqttOnLostConnect)
	opts.SetDefaultPublishHandler(MqttOnMessage)
	//设置Mqtt客户端用户名和密码
	opts.SetUsername(mBrokerUser.String())
	opts.SetPassword(mBrokerPassword.String())
	// 创建客户端
	MqttClient = mqtt.NewClient(opts)
}

// GetList 返回已注册程序列表
func GetList(r *ghttp.Request) {}

// GetAbout 返回版本信息
func GetAbout(r *ghttp.Request) {
	mHeader := NTPack.NewComponentHeader(1)
	//插入Components表，TODO 测试，正式运行时需要删除
	_, er4 := g.DB().Model("Components").Insert(gjson.New(mHeader))
	fmt.Println(mHeader, er4)
	//TODO 测试结束
	r.Response.WriteJson(mHeader)
}

// PostReg 接受新程序注册
func PostReg(r *ghttp.Request) {
	//读取客户端提交的注册参数
	m1, er := r.GetJson()
	if (m1 != nil) && (er != nil) {
		m1bytes, er2 := m1.ToJson()
		if er2 != nil {
			var m3 NTPack.TCBComponentHeader
			er3 := json.Unmarshal(m1bytes, &m3)
			if er3 != nil {
				//TODO 检查AccessKey是否匹配
				if m3.AccessKey == AccessKey {
					//匹配，允许接入
					//插入Components表
					_, er4 := g.DB().Model("Components").Insert(gjson.New(m3))
					fmt.Println(m3, er4)
					//返回注册成功和ENV参数
				} else {
					//不匹配，不允许接入
				}
			}
		}
	}
}
