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
	"github.com/gogf/gf/v2/os/gctx"
)

func main() {
	//server := g.Server()
	m, _ := g.Config().Get(gctx.New(), "custom.accessKey")
	fmt.Println(m)
	//server.Run()

}
