/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-07-29
 * Description:
 * This file is part of the NTPack project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年07月29日
 * 描述：通用函数
 --------------------------------------------------------*/

package NTPack

import (
	"fmt"
	"github.com/bwmarrin/snowflake"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"net"
	"os"
	"time"
)

// GetPid 获取进程ID
func GetPid() int {
	pid := os.Getpid()
	return pid
}

// GetLocalIP 获取本地非回环IPv4地址
func GetLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue // 跳过未启用或回环接口
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip != nil {
				return ip.String(), nil
			}
		}
	}
	return "", fmt.Errorf("未找到有效本地IP")
}

func GetSnowflake(anode int64) (int64, error) {
	// 创建节点（机器ID），范围0-1023
	node, err := snowflake.NewNode(anode)
	if err != nil {
		panic(err)
	}

	// 生成ID
	id := node.Generate()
	return id.Int64(), nil

}

// InitMqttClient 初始化Mqtt客户端
func InitMqttClient(AComponent TCBComponentHeader, OnConnectEvent mqtt.OnConnectHandler, OnLostConnectEvent mqtt.ConnectionLostHandler, OnMessageEvent mqtt.MessageHandler) mqtt.Client {
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
	opts.SetOnConnectHandler(OnConnectEvent)
	opts.SetConnectionLostHandler(OnLostConnectEvent)
	opts.SetDefaultPublishHandler(OnMessageEvent)
	//设置Mqtt客户端用户名和密码
	opts.SetUsername(mBrokerUser.String())
	opts.SetPassword(mBrokerPassword.String())
	// 创建客户端
	return mqtt.NewClient(opts)
}
