//---------------------------------------------
package framework

//---------------------------------------------
import (
	TIME "time"

	UTILS "FKGoServer/FKLib_Common/Utils"
	PROTO "FKGoServer/FKServer_Agent/Proto"
	SESSION "FKGoServer/FKServer_Agent/Session"
)

//---------------------------------------------
// 处理一个客户端会话的全部事务
func func_ClientSessionHandler(sess *SESSION.Session, in chan []byte, out *Buffer) {
	defer SYNC_WAITGROUP.Done()        // 无论如何，最终要删除一个引用，避免程序无法退出
	defer UTILS.Func_PrintPanicStack() // 无论如何，最重要打印引发异常的堆栈

	// 初始化会话
	sess.MQ = make(chan PROTO.Game_Frame, 512)
	sess.ConnectTime = TIME.Now()
	sess.LastPacketTime = TIME.Now()
	// 创建一分钟定时器消息
	min_timer := TIME.After(TIME.Minute)

	// 线程创建完毕，无论如何，最终要进行清理行为
	defer func() {
		close(sess.Die)
		if sess.Stream != nil {
			sess.Stream.CloseSend()
		}
	}()

	/* 主消息循环
	1： 负责接收客户端发来的消息
	2： 负责接收游戏服务器发来的消息
	3： 负责定时器
	4： 负责服务器关闭信号处理
	*/
	for {
		select {
		case msg, ok := <-in: // 从网络来的客户端消息
			if !ok {
				return
			}

			sess.PacketCount++
			sess.PacketTime = TIME.Now()

			if result := func_UserMsgHandler(sess, msg); result != nil {
				out.func_CreateAndSendMsgPacket(sess, result)
			}
			sess.LastPacketTime = sess.PacketTime

		case frame := <-sess.MQ: // 从游戏服务器来的消息
			switch frame.Type {
			case PROTO.Game_Message:
				out.func_CreateAndSendMsgPacket(sess, frame.Message)
			case PROTO.Game_Kick:
				sess.Flag |= SESSION.SESS_KICKED_OUT
			}

		case <-min_timer: // 一分钟定时器事件
			func_OnTimer_OneMinute(sess, out)
			min_timer = TIME.After(TIME.Minute)

		case <-DIE_SIGN: // 服务器关闭信号
			sess.Flag |= SESSION.SESS_KICKED_OUT
		}

		// 被标记本客户端必须被踢掉
		if sess.Flag&SESSION.SESS_KICKED_OUT != 0 {
			return
		}
	}
}

//---------------------------------------------
