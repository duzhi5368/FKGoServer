//---------------------------------------------
package msg

//---------------------------------------------
import (
	RC4 "crypto/rc4"
	FMT "fmt"
	IO "io"
	BIG "math/big"

	DH "FKGoServer/FKLib_Common/DH"
	SERVICES "FKGoServer/FKLib_Common/Service"
	PROTO "FKGoServer/FKServer_Agent/Proto"
	SESSION "FKGoServer/FKServer_Agent/Session"

	LOG "github.com/Sirupsen/logrus"
	CONTEXT "golang.org/x/net/context"
	METADATA "google.golang.org/grpc/metadata"

	MSGDEFINE "FKGoServer/FKLib_Common/MsgDefine"
	PACKET "FKGoServer/FKLib_Common/Packet"
)

//---------------------------------------------
// 声明消息分发类
var Handlers map[int16]func(*SESSION.Session, *PACKET.Packet) []byte

//---------------------------------------------
// 注册消息分发函数
func init() {
	Handlers = map[int16]func(*SESSION.Session, *PACKET.Packet) []byte{
		0:  P_heart_beat_req,
		10: P_user_login_req,
		30: P_get_seed_req,
	}
}

//---------------------------------------------
// 心跳包
func P_heart_beat_req(sess *SESSION.Session, reader *PACKET.Packet) []byte {
	tbl, _ := MSGDEFINE.PKT_auto_id(reader)
	return PACKET.Func_Pack(MSGDEFINE.Code["heart_beat_ack"], tbl, nil)
}

//---------------------------------------------
// 密钥交换
// 加密建立方式: DH+RC4
// 注意:完整的加密过程包括 RSA+DH+RC4
// 1. RSA用于鉴定服务器的真伪(这步省略)
// 2. DH用于在不安全的信道上协商安全的KEY
// 3. RC4用于流加密
func P_get_seed_req(sess *SESSION.Session, reader *PACKET.Packet) []byte {
	tbl, _ := MSGDEFINE.PKT_seed_info(reader)
	// KEY1
	X1, E1 := DH.DHExchange()
	KEY1 := DH.DHKey(X1, BIG.NewInt(int64(tbl.F_client_send_seed)))

	// KEY2
	X2, E2 := DH.DHExchange()
	KEY2 := DH.DHKey(X2, BIG.NewInt(int64(tbl.F_client_receive_seed)))

	ret := MSGDEFINE.S_seed_info{int32(E1.Int64()), int32(E2.Int64())}
	// 服务器加密种子是客户端解密种子
	encoder, err := RC4.NewCipher([]byte(FMT.Sprintf("%v%v", MSGDEFINE.SALT, KEY2)))
	if err != nil {
		LOG.Error(err)
		return nil
	}
	decoder, err := RC4.NewCipher([]byte(FMT.Sprintf("%v%v", MSGDEFINE.SALT, KEY1)))
	if err != nil {
		LOG.Error(err)
		return nil
	}
	sess.Encoder = encoder
	sess.Decoder = decoder
	sess.Flag |= SESSION.SESS_KEYEXCG
	return PACKET.Func_Pack(MSGDEFINE.Code["get_seed_ack"], ret, nil)
}

//---------------------------------------------
// 玩家登陆过程
func P_user_login_req(sess *SESSION.Session, reader *PACKET.Packet) []byte {
	// TODO: 登陆鉴权
	// 简单鉴权可以在agent直接完成，通常公司都存在一个用户中心服务器用于鉴权
	sess.UserId = 1

	// TODO: 选择GAME服务器
	// 选服策略依据业务进行，比如小服可以固定选取某台，大服可以采用HASH或一致性HASH
	sess.GSID = MSGDEFINE.DEFAULT_GSID

	// 连接到已选定GAME服务器
	conn := SERVICES.GetServiceWithId("game-10000", sess.GSID)
	if conn == nil {
		LOG.Error("cannot get game service:", sess.GSID)
		return nil
	}
	cli := PROTO.NewGameServiceClient(conn)

	// 开启到游戏服的流
	ctx := METADATA.NewContext(CONTEXT.Background(), METADATA.New(map[string]string{"userid": FMT.Sprint(sess.UserId)}))
	stream, err := cli.Stream(ctx)
	if err != nil {
		LOG.Error(err)
		return nil
	}
	sess.Stream = stream

	// 读取GAME返回消息的goroutine
	fetcher_task := func(sess *SESSION.Session) {
		for {
			in, err := sess.Stream.Recv()
			if err == IO.EOF { // 流关闭
				LOG.Debug(err)
				return
			}
			if err != nil {
				LOG.Error(err)
				return
			}
			select {
			case sess.MQ <- *in:
			case <-sess.Die:
			}
		}
	}
	go fetcher_task(sess)
	return PACKET.Func_Pack(MSGDEFINE.Code["user_login_succeed_ack"], MSGDEFINE.S_user_snapshot{F_uid: sess.UserId}, nil)
}

//---------------------------------------------
