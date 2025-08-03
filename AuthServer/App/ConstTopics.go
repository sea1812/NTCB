/**--------------------------------------------------------
 * Author: jiang5630@outlook.com 2025-07-31
 * Description:
 * This file is part of the AuthServer project.
 * --------------------------------------------------------
 * 作者：jiang5630@outlook.com  2025年07月31日
 * 描述：Topic常量定义
 --------------------------------------------------------*/

package NTPack

const (
	C_Public_Root_Topic   = "ntcb/"       //根频道
	C_Public_Enter_Topic  = "ntcb/Enter"  //组件上线通知频道，在此频道广播组件上线消息
	C_Public_Exit_Topic   = "ntcb/Exit"   //组件下线通知频道，在此频道广播组件下线消息
	C_Public_Ticker_Topic = "ntcb/Ticker" //时钟节拍通知频道，在此频道广播节拍消息
	C_Public_Notice_Topic = "ntcb/Notice" //公共通知下发频道，在此频道广播公共通知消息
	C_Public_Log_Topic    = "ntcb/Log"    //日志通知频道，在此频道广播日志消息，由后台服务接收消息并写入数据库
	C_Public_Stat_Topic   = "ntcb/Stat"   //组件状态通知频道，组件在此频道广播运行状态消息
	C_Daemon_Topic        = "ntcb/Daemon" //Daemon专属频道，针对Daemon指令在此频道广播下发

	//Daemon Command 命令字常量
	CMD_DAEMON_START_BOT = "daemon_start_bot"
)
