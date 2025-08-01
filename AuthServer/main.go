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
	"context"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/gogf/gf/contrib/drivers/pgsql/v2"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcron"
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
	SnowID       int64

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
	SnowID = mHeader.SnowID
	//清理Components表
	//er := g.DB().Model("Components").Data("Enable", 0).Where("Enable", 1)
	_, _ = g.DB().Model("Components").Update("CompEnabled=0", "CompEnabled=1")
	mJson := gjson.New(mHeader)
	//将自身Header数据插入Components表
	_, _ = g.DB().Model("Components").Insert(mJson)

	//创建MQTT客户端
	//InitMqttClient(*mHeader)
	MqttClient = NTPack.InitMqttClient(*mHeader, MqttOnConnect, MqttOnLostConnect, MqttOnMessage)
	fmt.Println("Testing message server...")
	//检查消息服务器是否可用，如果不可用则退出进程（当然这需要花费超时和重试的时间后才能触发异常退出）
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

	//设置定时任务，发送STAT广播，默认为每10分钟
	mPattern, _ := g.Config().Get(mCtx, "ntcb.statCronPattern")
	mPatternString := mPattern.String()
	if mPatternString == "" {
		mPatternString = "# */10 * * * *"
	}
	_, _ = gcron.Add(mCtx, mPatternString, func(ctx context.Context) {
		mStat := NTPack.TCBComponentStat{
			ComponentID: ComponentId,
			SnowID:      SnowID,
			StatMessage: "",
			StatCode:    "200",
			StatTime:    time.Now(),
		}
		MqttClient.Publish(NTPack.C_Public_Stat_Topic, 0, false, gjson.New(mStat).String())
	}, "AuthServerStat")
	//启动定时任务
	gcron.Start("AuthServerStat")
	//广播Public/Enter消息Auth服务上线
	mHeader.AccessKey = "hidden"
	MqttClient.Publish(NTPack.C_Public_Enter_Topic, 0, false, gjson.New(mHeader).String())
	//TODO 同时发布到日志频道
	mLog := NTPack.TCBComponentLog{
		ComponentID: ComponentId,
		SnowID:      SnowID,
		Type:        "service",
		LogMessage:  "AuthServerStart",
		Category:    "AuthServer",
	}
	MqttClient.Publish(NTPack.C_Public_Log_Topic, 0, false, gjson.New(mLog).String())
	s.Run()

}

func MqttOnConnect(client mqtt.Client) {
	//fmt.Println("Connected")
	//订阅频道
	MqttClient.Subscribe("ntcb/#", 0, nil)
}

func MqttOnLostConnect(client mqtt.Client, err error) {
	//fmt.Println("LostConnect")
}

func MqttOnMessage(client mqtt.Client, Message mqtt.Message) {
	//fmt.Println("OnMessage___________________")
	//fmt.Println(Message.Topic())
	//fmt.Println(string(Message.Payload()))
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
	if (m1 != nil) && (er == nil) {
		m1bytes, er2 := m1.ToJson()
		if er2 == nil {
			var m3 NTPack.TCBComponentHeader
			er3 := json.Unmarshal(m1bytes, &m3)
			if er3 == nil {
				//TODO 检查AccessKey是否匹配
				if m3.AccessKey == AccessKey {
					//匹配，允许接入
					//插入Components表
					_, _ = g.DB().Model("Components").Insert(gjson.New(m1))

					//返回注册成功
					r.Response.WriteJson(g.Map{
						"code":    200,
						"message": "register success",
					})
				} else {
					//不匹配，不允许接入
					r.Response.WriteJson(g.Map{
						"code":    404,
						"message": "register fail",
					})
				}
			} else {
				r.Response.WriteJson(g.Map{
					"code":    502,
					"message": fmt.Sprint(er3),
				})
			}
		} else {
			r.Response.WriteJson(g.Map{
				"code":    501,
				"message": fmt.Sprint(er2),
			})
		}
	} else {
		r.Response.WriteJson(g.Map{
			"code":    500,
			"message": fmt.Sprintln(er),
		})
	}
}
