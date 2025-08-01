/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-08-01
 * Description:
 * This file is part of the BotDaemon project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年08月01日
 * 描述：控制Bot进程创建的Daemon程序
 --------------------------------------------------------*/

package main

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
	mHeader := NTPack.NewComponentHeader(mServerIDInt)
	//向AuthServer申请注册
	Result, er := g.Client().Post(mCtx, mAuthServerString, gjson.New(mHeader).String())
	if er == nil {
		mResult := gjson.New(Result.ReadAllString()).Map()
		fmt.Println(mResult)
		if gconv.Int(mResult["code"]) == 200 {
			//注册成功，继续
			//创建Mqtt客户端

			//连接到Broker
			//订阅指令信道
			//发布上线通报
			//设置定时任务，发布STAT通报
			//进入循环，等待退出信号
			//退出信号触发，发出离线消息
		}
	} else {
		panic(er)
	}
}
