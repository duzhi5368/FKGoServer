//---------------------------------------------
package framework

//---------------------------------------------
import (
	TIME "time"

	MSGDEFINE "FKGoServer/FKLib_Common/MsgDefine"
	PACKET "FKGoServer/FKLib_Common/Packet"
	UTILS "FKGoServer/FKLib_Common/Utils"
	MSG "FKGoServer/FKServer_Agent/Msg"
	SESSION "FKGoServer/FKServer_Agent/Session"

	LOG "github.com/Sirupsen/logrus"
)

//---------------------------------------------
// 客户端消息处理代理
func func_UserMsgHandler(sess *SESSION.Session, p []byte) []byte {
	// 记录当前时间
	start := TIME.Now()
	// 无论如何，最终打印引发错误的日志
	defer UTILS.Func_PrintPanicStack(sess, p)
	// 解密
	if sess.Flag&SESSION.SESS_ENCRYPT != 0 {
		sess.Decoder.XORKeyStream(p, p)
	}
	// 封装为reader
	reader := PACKET.Reader(p)

	// 读客户端数据包序列号(1,2,3...)
	// 客户端发送的数据包必须包含一个自增的序号，必须严格递增
	// 加密后，可避免重放攻击-REPLAY-ATTACK
	seq_id, err := reader.ReadU32()
	if err != nil {
		LOG.Error("读取客户端数据包序列号失败:", err)
		sess.Flag |= SESSION.SESS_KICKED_OUT
		return nil
	}

	// 数据包序列号验证
	if seq_id != sess.PacketCount {
		LOG.Errorf("数据包序列号错误 实际包ID:%v 期望包ID:%v 包大小:%v", seq_id, sess.PacketCount, len(p)-6)
		sess.Flag |= SESSION.SESS_KICKED_OUT
		return nil
	}

	// 读协议号
	b, err := reader.ReadS16()
	if err != nil {
		LOG.Error("读取协议号失败.")
		sess.Flag |= SESSION.SESS_KICKED_OUT
		return nil
	}

	// 根据协议号断做服务划分
	// 协议号的划分采用分割协议区间, 用户可以自定义多个区间，用于转发到不同的后端服务
	var ret []byte
	if b > CONST_GameServerMsgIDBegin {
		if err := func_ForwardMsgToGameServer(sess, p[4:]); err != nil {
			LOG.Errorf("服务 ID:%v 执行失败, 错误信息:%v", b, err)
			sess.Flag |= SESSION.SESS_KICKED_OUT
			return nil
		}
	} else {
		if h := MSG.Handlers[b]; h != nil {
			ret = h(sess, reader)
		} else {
			LOG.Errorf("服务 ID:%v 没有注册处理函数", b)
			sess.Flag |= SESSION.SESS_KICKED_OUT
			return nil
		}
	}

	elasped := TIME.Now().Sub(start)
	if b != 0 { // 排除心跳包日志
		LOG.WithFields(LOG.Fields{"消耗时间": elasped,
			"接口": MSGDEFINE.RCode[b],
			"编号": b}).Debug("REQ")
	}
	return ret
}

//---------------------------------------------
