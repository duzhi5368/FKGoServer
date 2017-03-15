//---------------------------------------------
package framework

//---------------------------------------------
import (
	TIME "time"

	SESSION "FKGoServer/FKServer_Agent/Session"

	LOG "github.com/Sirupsen/logrus"
)

//---------------------------------------------
// 客户端1分钟定时器
func func_OnTimer_OneMinute(sess *SESSION.Session, out *Buffer) {
	// 客户端登陆时间
	interval := TIME.Now().Sub(sess.ConnectTime).Minutes()

	if interval >= 1 { // 登录时长超过1分钟才开始统计rpm。防脉冲
		rpm := float64(sess.PacketCount) / interval

		// 发包频率控制，太高的RPM直接踢掉
		if rpm > CONST_RpmLimit {
			sess.Flag |= SESSION.SESS_KICKED_OUT
			LOG.WithFields(LOG.Fields{
				"userid": sess.UserId,
				"rpm":    rpm,
			}).Error("RPM")
			return
		}
	}
}

//---------------------------------------------
