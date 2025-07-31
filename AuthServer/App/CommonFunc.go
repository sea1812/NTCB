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
	"net"
	"os"
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
