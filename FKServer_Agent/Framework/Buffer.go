//---------------------------------------------
package framework

//---------------------------------------------
import (
	BINARY "encoding/binary"
	NET "net"

	PACKET "FKGoServer/FKLib_Common/Packet"
	UTILS "FKGoServer/FKLib_Common/Utils"
	SESSION "FKGoServer/FKServer_Agent/Session"

	LOG "github.com/Sirupsen/logrus"
)

//---------------------------------------------
// 发送给客户端的数据包对象
type Buffer struct {
	ctrl    chan struct{} // 用以接收exit消息
	pending chan []byte   // 实际需要发送的数据缓冲
	conn    NET.Conn      // 网络连接对象
	cache   []byte        // 留作系统写入的缓冲
}

//---------------------------------------------
// 组包并压入发送栈
func (buf *Buffer) func_CreateAndSendMsgPacket(sess *SESSION.Session, data []byte) {
	// 如果需要发送的数据为空
	if data == nil {
		return
	}

	// 进行加密
	// (NOT_ENCRYPTED) -> KEYEXCG -> ENCRYPT
	if sess.Flag&SESSION.SESS_ENCRYPT != 0 { // 开启加密
		sess.Encoder.XORKeyStream(data, data)
	} else if sess.Flag&SESSION.SESS_KEYEXCG != 0 { // Key发生更变，当前还不能进行加密
		sess.Flag &^= SESSION.SESS_KEYEXCG
		sess.Flag |= SESSION.SESS_ENCRYPT
	}

	// 将数据压送到发送队列中
	buf.pending <- data
	return
}

//---------------------------------------------
// 实际发包协程
func (buf *Buffer) func_StartSendPacket() {
	defer UTILS.Func_PrintPanicStack()
	for {
		select {
		case data := <-buf.pending:
			buf.func_EncapsulationAndSendPacket(data)
		case <-buf.ctrl: // 接收到会话结束消息
			close(buf.pending)
			// 关闭本连接
			buf.conn.Close()
			return
		}
	}
}

//---------------------------------------------
// 进行包封装并发送
func (buf *Buffer) func_EncapsulationAndSendPacket(data []byte) bool {
	// 进行包组装
	sz := len(data)
	BINARY.BigEndian.PutUint16(buf.cache, uint16(sz))
	copy(buf.cache[2:], data)

	// 写入包
	n, err := buf.conn.Write(buf.cache[:sz+2])
	if err != nil {
		LOG.Warningf("写入发送包失败, 写入大小: %v 错误原因: %v", n, err)
		return false
	}

	return true
}

//---------------------------------------------
// 为一个会话创建一个写入缓冲区
func func_CreateWriteBuffer(conn NET.Conn, ctrl chan struct{}) *Buffer {
	buf := Buffer{conn: conn}
	buf.pending = make(chan []byte)
	buf.ctrl = ctrl
	buf.cache = make([]byte, PACKET.PACKET_LIMIT+2)
	return &buf
}

//---------------------------------------------
