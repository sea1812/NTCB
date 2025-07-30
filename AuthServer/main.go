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
	mHeader := NTPack.NewComponentHeader(1)
	//插入Components表，TODO 测试，正式运行时需要删除
	_, er4 := g.DB().Model("Components").Insert(gjson.New(mHeader))
	fmt.Println(mHeader, er4)
	//
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
				//插入Components表
				_, er4 := g.DB().Model("Components").Insert(gjson.New(m3))
				fmt.Println(m3, er4)
				//返回注册成功和ENV参数
				//广播重新注册消息，预防因AuthServer中途退出或重启而丢失设备

			}
		}
	}
}
