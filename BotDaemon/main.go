/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-08-01
 * Description:
 * This file is part of the BotDaemon project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年08月01日
 * 描述：控制Bot进程创建的Daemon程序
 --------------------------------------------------------*/

package main

import "C"
import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/util/gconv"
	NTPack "github.com/sea1812/NTCB/AuthServer/App"
)

var (
	MqttClient mqtt.Client
	CompHeader *NTPack.TCBComponentHeader //头
)

func main() {
	//获取Config中的参数
	mCtx := gctx.New()
	//获取注册参数
	mAuthServer, _ := g.Config().Get(mCtx, "ntcb.authServer")
	mAuthServerString := mAuthServer.String() + "/reg"
	mServerID, _ := g.Config().Get(mCtx, "serverID")
	mServerIDInt := mServerID.Int64()
	//生成ComponentHeader
	CompHeader := NTPack.NewComponentHeader(mServerIDInt)
	//向AuthServer申请注册
	fmt.Println("Registering...")
	Result, er := g.Client().Post(mCtx, mAuthServerString, gjson.New(CompHeader).String())
	if er == nil {
		mResult := gjson.New(Result.ReadAllString()).Map()
		if gconv.Int(mResult["code"]) == 200 {
			//注册成功，继续
			fmt.Println("Register success. Connecting message broker...")
			//创建Mqtt客户端
			MqttClient = NTPack.InitMqttClient(*CompHeader, MqttOnConnect, MqttOnLostConnect, MqttOnMessage)
			//连接到Broker
			//检查消息服务器是否可用，如果不可用则退出进程（当然这需要花费超时和重试的时间后才能触发异常退出）
			if token := MqttClient.Connect(); token.Wait() && token.Error() != nil {
				panic(token.Error())
			}
			defer MqttClient.Disconnect(250) // 优雅断开连接，等待250ms处理剩余消息
			//在OnConnectEvent中订阅指令信道
			//发布上线通报
			//广播Public/Enter消息Auth服务上线
			CompHeader.AccessKey = "hidden"
			MqttClient.Publish(NTPack.C_Public_Enter_Topic, 0, false, gjson.New(CompHeader).String())
			//设置定时任务，发布STAT通报
			//进入循环，等待退出信号
			//退出信号触发，发出离线消息
		}
	} else {
		panic(er)
	}
}

func MqttOnConnect(client mqtt.Client) {
	//fmt.Println("Connected")
	//TODO 订阅频道
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
