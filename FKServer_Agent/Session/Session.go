//---------------------------------------------
package Session

//---------------------------------------------
import (
	PROTO "FKGoServer/FKServer_Agent/Proto"
	RC4 "crypto/rc4"
	NET "net"
	TIME "time"
)

//---------------------------------------------
const (
	SESS_KEYEXCG    = 0x1 // 是否已经交换完毕KEY
	SESS_ENCRYPT    = 0x2 // 是否可以开始加密
	SESS_KICKED_OUT = 0x4 // 踢掉
)

//---------------------------------------------
type Session struct {
	IP      NET.IP                         // 客户端IP
	MQ      chan PROTO.Game_Frame          // 返回给客户端的异步消息
	Encoder *RC4.Cipher                    // 加密器
	Decoder *RC4.Cipher                    // 解密器
	UserId  int32                          // 玩家ID
	GSID    string                         // 游戏服ID;e.g.: game1,game2
	Stream  PROTO.GameService_StreamClient // 后端游戏服数据流
	Die     chan struct{}                  // 会话关闭信号

	Flag int32 // 会话标记

	ConnectTime    TIME.Time // TCP链接建立时间
	PacketTime     TIME.Time // 当前包的到达时间
	LastPacketTime TIME.Time // 前一个包到达时间

	PacketCount uint32 // 对收到的包进行计数，避免恶意发包
}
