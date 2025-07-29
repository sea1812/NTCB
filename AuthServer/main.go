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
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"time"
)

var (
	// 声明全局变量
	AccessKey   string
	PublisherId string
	ComponentId string
	StartTime   time.Time
)

func main() {
	//填充全局变量
	mCtx := gctx.New()
	mAccessKey, _ := g.Config().Get(mCtx, "ntcb.accessKey")
	mPublisherId, _ := g.Config().Get(mCtx, "ntcb.publisherId")
	mComponentId, _ := g.Config().Get(mCtx, "ntcb.componentId")
	AccessKey = mAccessKey.String()
	PublisherId = mPublisherId.String()
	ComponentId = mComponentId.String()
	fmt.Println(AccessKey, PublisherId, ComponentId) //TODO 回头删掉
	//创建MQTT客户端

	//TODO 检查消息服务器是否可用，如果不可用则启动消息服务器

	//设置Http路径
	s := g.Server()
	mGroup := s.Group("/")
	mGroup.GET("/about", GetAbout) //返回版本信息
	mGroup.GET("/list", GetList)   //返回已注册程序列表
	mGroup.POST("/reg", PostReg)   //注册新设备

	//TODO 获取进程ID、IP等数据
	//TODO 广播Public/Enter消息Auth服务上线，同时发布到日志频道
	StartTime = time.Now()
	s.Run()
}

// GetList 返回已注册程序列表
func GetList(r *ghttp.Request) {}

// GetAbout 返回版本信息
func GetAbout(r *ghttp.Request) {
	mCtx := gctx.New()
	mVersion, _ := g.Config().Get(mCtx, "ntcb.appVersion")
	mIntro, _ := g.Config().Get(mCtx, "ntcb.appIntro")
	mAuthor, _ := g.Config().Get(mCtx, "ntcb.appAuthor")
	r.Response.WriteJson(g.Map{
		"version":   mVersion.String(),
		"intro":     mIntro.String(),
		"author":    mAuthor.String(),
		"startTime": StartTime.Format("2006-01-02 15:04:05"),
	})
}

// PostReg 接受新程序注册
func PostReg(r *ghttp.Request) {}
