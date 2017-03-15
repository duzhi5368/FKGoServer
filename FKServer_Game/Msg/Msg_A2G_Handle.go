//---------------------------------------------
package msg

//---------------------------------------------
import (
	MSGDEFINE "FKGoServer/FKLib_Common/MsgDefine"
	PACKET "FKGoServer/FKLib_Common/Packet"
	SESSION "FKGoServer/FKServer_Game/Session"
)

//---------------------------------------------
// 消息处理回调组
var Handlers map[int16]func(*SESSION.Session, *PACKET.Packet) []byte

//---------------------------------------------
// 注册消息
func init() {
	Handlers = map[int16]func(*SESSION.Session, *PACKET.Packet) []byte{
		1001: P_proto_ping_req,
	}
}

//---------------------------------------------
// ping消息的回调处理
func P_proto_ping_req(sess *SESSION.Session, reader *PACKET.Packet) []byte {
	tbl, _ := MSGDEFINE.PKT_auto_id(reader)
	return PACKET.Func_Pack(MSGDEFINE.Code["proto_ping_ack"], tbl, nil)
}

//---------------------------------------------
