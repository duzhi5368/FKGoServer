//---------------------------------------------
package framework

//---------------------------------------------
import (
	ERRORS "errors"

	PROTO "FKGoServer/FKServer_Agent/Proto"
	SESSION "FKGoServer/FKServer_Agent/Session"

	LOG "github.com/Sirupsen/logrus"
)

//---------------------------------------------
// 向游戏服务器推送消息
func func_ForwardMsgToGameServer(sess *SESSION.Session, p []byte) error {
	frame := &PROTO.Game_Frame{
		Type:    PROTO.Game_Message,
		Message: p,
	}

	// 检查流
	if sess.Stream == nil {
		return ERRORS.New("尚未开启流")
	}

	// 推送消息帧给GameServer
	if err := sess.Stream.Send(frame); err != nil {
		LOG.Error(err)
		return err
	}
	return nil
}

//---------------------------------------------
