/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-08-03
 * Description:
 * This file is part of the BotDolly project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年08月03日
 * 描述：第一个Bot Dolly，用于测试模版
 --------------------------------------------------------*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/util/gconv"
	NTPack "github.com/sea1812/NTCB/AuthServer/App"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	//全局变量定义
	G_AutoShutdown bool = true //是否运行完Job自动退出
	G_AUtoStartJob bool = true //是否自动运行Job
	MqttClient     mqtt.Client
	CompHeader     *NTPack.TCBComponentHeader //头

)

func main() {
	var done chan bool = make(chan bool, 1)
	mCtx := gctx.New()
	//读取Config中的参数设置
	mAutoShutdown, _ := g.Config("botdolly").Get(mCtx, "ntcb.autoshutdown")
	G_AutoShutdown = mAutoShutdown.Bool()
	mAutoStartJob, _ := g.Config("botdolly").Get(mCtx, "ntcb.autostartjob")
	G_AUtoStartJob = mAutoStartJob.Bool()
	//如果设置了不自动退出，则生成系统信号
	if G_AutoShutdown == false {
		sigs := make(chan os.Signal, 1)
		done = make(chan bool, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			done <- true
		}()
	}
	//获取注册参数
	mAuthServer, _ := g.Config("botdolly").Get(mCtx, "ntcb.authServer")
	mAuthServerString := mAuthServer.String() + "/reg"
	mServerID, _ := g.Config("botdolly").Get(mCtx, "serverID")
	mServerIDInt := mServerID.Int64()
	//生成ComponentHeader
	CompHeader = NTPack.NewComponentHeader(mServerIDInt, "botdolly")
	//向Auth Server申请注册
	fmt.Println("Registering...")
	Result, er := g.Client().Post(mCtx, mAuthServerString, gjson.New(CompHeader).String())
	if er == nil {
		mResult := gjson.New(Result.ReadAllString()).Map()
		if gconv.Int(mResult["code"]) == 200 {
			//注册成功，继续
			fmt.Println("Register success. Connecting message broker...")
			//创建Mqtt客户端
			MqttClient = NTPack.InitMqttClient(*CompHeader, MqttOnConnect, MqttOnLostConnect, MqttOnMessage, "botdolly")
			//连接到Broker
			//检查消息服务器是否可用，如果不可用则退出进程（当然这需要花费超时和重试的时间后才能触发异常退出）
			if token := MqttClient.Connect(); token.Wait() && token.Error() != nil {
				panic(token.Error())
			}
			defer MqttClient.Disconnect(250) // 优雅断开连接，等待250ms处理剩余消息
			//在OnConnectEvent中订阅指令信道
			//发布上线通报，广播Public/Enter消息Auth服务上线
			CompHeader.AccessKey = "hidden"
			MqttClient.Publish(NTPack.C_Public_Enter_Topic, 0, false, gjson.New(CompHeader).String())
			//设置定时任务，发送STAT广播，默认为每10分钟
			mPattern, _ := g.Config("botdolly").Get(mCtx, "ntcb.statCronPattern")
			mPatternString := mPattern.String()
			if mPatternString == "" {
				mPatternString = "# */10 * * * *"
			}
			_, _ = gcron.Add(mCtx, mPatternString, func(ctx context.Context) {
				mStat := NTPack.TCBComponentStat{
					ComponentID: CompHeader.ComponentID,
					SnowID:      CompHeader.SnowID,
					StatMessage: "",
					StatCode:    "200",
					StatTime:    time.Now(),
				}
				MqttClient.Publish(NTPack.C_Public_Stat_Topic, 0, false, gjson.New(mStat).String())
			}, "BotStat")
			//启动定时任务
			gcron.Start("BotStat")
			//广播Public/Enter消息Auth服务上线
			CompHeader.AccessKey = "hidden"
			MqttClient.Publish(NTPack.C_Public_Enter_Topic, 0, false, gjson.New(CompHeader).String())
			//开始等候MQTT消息启动动作
			fmt.Println("Bot is running.")
			//如果是自动执行JOB，则启动JOB
			if G_AUtoStartJob == true {
				go DoBotJob() //在协程中启动Job
			}
			//发送JOB启动消息
			//JOB完成，发送JOB DONE消息
		}
	} else {
		panic(er)
	}
	//如果设置了AutoShutdown则退出，否则等待信号
	if G_AutoShutdown == false {
		//进入循环，等待退出信号
		go func() {
		}()

		<-done

	}
	//退出信号触发，发出离线消息
	CompHeader.AccessKey = "hidden"
	MqttClient.Publish(NTPack.C_Public_Exit_Topic, 0, false, gjson.New(CompHeader).String())
	fmt.Println("Bot exited.")
}

// DoBotJob 运行任务
func DoBotJob() {
	//发送JobStart消息
	//Dolly的唯一功能是在专属信道广播Mie Mie
	for i := 0; i < 3; i++ {
		MqttClient.Publish(NTPack.C_Bot_Topic+"/"+fmt.Sprint(CompHeader.SnowID)+"/", 0, false, "Mie Mie, I am dolly")
		fmt.Println("Dolly braying...", i)
		time.Sleep(3 * time.Second)
	}
	//发送JobDone消息
}

func MqttOnConnect(client mqtt.Client) {
	//fmt.Println("Connected")
	//TODO 订阅频道
	//MqttClient.Subscribe("ntcb/#", 0, nil)
	//订阅Bot专属频道
	MqttClient.Subscribe(NTPack.C_Bot_Topic+"/"+fmt.Sprint(CompHeader.SnowID), 0, nil)
}

func MqttOnLostConnect(client mqtt.Client, err error) {
	//fmt.Println("LostConnect")
}

func MqttOnMessage(client mqtt.Client, Message mqtt.Message) {
	//处理Bot专属频道来的消息，主要是启动Job命令
	fmt.Println(string(Message.Payload()))
	var mCmd NTPack.TCBBotCommand
	m1, er1 := gjson.New(Message.Payload()).ToJson()
	er := json.Unmarshal(m1, &mCmd)
	fmt.Println(er1, er)
	fmt.Println(gjson.New(mCmd).String())
	if er == nil {
		fmt.Println(mCmd.CommandKey)
		if mCmd.CommandKey == NTPack.CMD_Bot_Start_Job {
			go DoBotJob()
		}
	}
}
