//---------------------------------------------
package framework

//---------------------------------------------
const (
	CONST_ReadDeadline         = 15       // 秒(没有网络包进入的最大间隔)
	CONST_ReceiveBuffer        = 32767    // 每个连接的接收缓冲区
	CONST_SendBuffer           = 65535    // 每个连接的发送缓冲区
	CONST_UdpBuffer            = 16777216 // UDP监听器的缓冲区
	CONST_TosEF                = 46       // Expedited Forwarding (EF)
	CONST_RpmLimit             = 200      // 每分钟许可的最大请求数
	CONST_ListerPort           = ":8888"  // TCP/UDP监听连接端口
	CONST_GameServerMsgIDBegin = 1000     // 转发给游戏服务器的MsgID起始值
)

//---------------------------------------------
