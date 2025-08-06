/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-07-29
 * Description:
 * This file is part of the NTPack project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年07月29日
 * 描述：通用类型定义
 --------------------------------------------------------*/

package NTPack

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"time"
)

// TCBComponentHeader 组件头信息结构，用于标识自身和组装报文
type TCBComponentHeader struct {
	ComponentID  string    `json:"componentID"`  //组件名
	PublisherID  string    `json:"publisherID"`  //发布消息的署名
	Version      string    `json:"version"`      //版本号
	Intro        string    `json:"intro"`        //介绍
	Author       string    `json:"author"`       //作者名称
	StartTime    time.Time `json:"startTime"`    //启动时间
	SnowID       int64     `json:"snowID"`       //雪花ID
	ServerNodeID int64     `json:"serverNodeID"` //服务器节点ID
	Pid          int       `json:"pid"`          //进程ID
	LocalIP      string    `json:"localIP"`      //本地IP
	AccessKey    string    `json:"accessKey"`    //Access Key
	CompEnabled  int       `json:"compEnabled"`  //是否启用
}

// NewComponentHeader 便捷命令，创建程序头
func NewComponentHeader(AServerNodeId int64, Config string) *TCBComponentHeader {
	a := new(TCBComponentHeader)
	mCtx := gctx.New()
	mComponentID, _ := g.Config(Config).Get(mCtx, "ntcb.componentId")
	mPublisherId, _ := g.Config(Config).Get(mCtx, "ntcb.publisherId")
	mVersion, _ := g.Config(Config).Get(mCtx, "ntcb.appVersion")
	mIntro, _ := g.Config(Config).Get(mCtx, "ntcb.appIntro")
	mAuthor, _ := g.Config(Config).Get(mCtx, "ntcb.appAuthor")
	mAccessKey, _ := g.Config(Config).Get(mCtx, "ntcb.accessKey")

	a.ComponentID = mComponentID.String()
	a.PublisherID = mPublisherId.String()
	a.Version = mVersion.String()
	a.Intro = mIntro.String()
	a.Author = mAuthor.String()
	a.AccessKey = mAccessKey.String()
	a.Pid = GetPid()
	a.LocalIP, _ = GetLocalIP()
	a.ServerNodeID = AServerNodeId
	a.SnowID, _ = GetSnowflake(AServerNodeId)
	a.StartTime = time.Now()
	a.CompEnabled = 1

	return a
}

// TCBComponentStat 组件状态信息结构，用于生成广播报文
type TCBComponentStat struct {
	ComponentID string    `json:"componentID"` //组件ID
	SnowID      int64     `json:"snowID"`      //雪花ID
	StatCode    string    `json:"statCode"`    //状态码
	StatMessage string    `json:"statMessage"` //状态消息
	StatTime    time.Time `json:"statTime"`    //报告状态的时间
}

// TCBComponentLog 日志信息结构，用于生成日志报文
type TCBComponentLog struct {
	ComponentID string    `json:"componentID"` //组件ID
	SnowID      int64     `json:"snowID"`      //雪花ID
	LogTime     time.Time `json:"logTime"`     //日志时间
	Category    string    `json:"category"`    //日志分类，如Auth Server
	Type        string    `json:"type"`        //日志类型，如Service、Daemon、Bot
	LogMessage  string    `json:"logMessage"`  //日志内容
}

// TCBDaemonCommand Daemon频道广播的命令消息结构
type TCBDaemonCommand struct {
	PublisherID      string    `json:"publisherID"`      //发送者的ComponentID
	PublishTime      time.Time `json:"publishTime"`      //发送时间
	CommandID        string    `json:"commandID"`        //命令ID
	DaemonSnowID     int64     `json:"daemonSnowID"`     //接受者只能是Daemon，这里填写他的SnowID
	CommandKey       string    `json:"command"`          //命令字
	BotID            string    `json:"botID"`            //作用对象Bot的ComponentID
	CommandString    string    `json:"commandString"`    //启动命令行，作为创建进程时用
	CommandArguments string    `json:"commandArguments"` //启动命令行的参数
}

// TCBDaemonCommandReceipt 通过Daemon/Receipt频道发送的命令执行回执
type TCBDaemonCommandReceipt struct {
	CommandID   string    `json:"commandID"`   //发送的CommandID
	PublishTime time.Time `json:"publishTime"` //回执时间
	Code        int       `json:"code"`        //状态码
	Message     string    `json:"message"`     //信息文本
}

// TCBBotJobStart 在Bot专属频道广播的JobStart消息结构
type TCBBotJobStart struct {
}

// TCBBotJobDone 在Bot专属频道广播的JobDone消息结构
type TCBBotJobDone struct {
}

// TCBBotCommand Bot专属频道广播的命令消息结构
type TCBBotCommand struct {
	CommandID   string    `json:"commandID"`   //发送的命令ID
	PublishTime time.Time `json:"publishTime"` //发送时间
	CommandKey  string    `json:"commandKey"`  //命令字
	Arguments   string    `json:"arguments"`   //参数
}

// TCBBotCommandReceipt 通过Bot专属频道发送的命令回执,在Bot/{SnowID}/Receipt频道发送
type TCBBotCommandReceipt struct {
	CommandID   string    `json:"commandID"`
	PublishTime time.Time `json:"publishTime"`
	Code        int       `json:"code"`
	Message     string    `json:"message"`
}
