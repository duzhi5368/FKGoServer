//---------------------------------------------
package framework

//---------------------------------------------
import (
	ERRORS "errors"
	IO "io"
	STRCONV "strconv"

	METADATA "google.golang.org/grpc/metadata"

	LOG "github.com/Sirupsen/logrus"

	PACKET "FKGoServer/FKLib_Common/Packet"
	UTILS "FKGoServer/FKLib_Common/Utils"
	LOGIC "FKGoServer/FKServer_Game/Logic"
	MSG "FKGoServer/FKServer_Game/Msg"
	PROTO "FKGoServer/FKServer_Game/Proto"
	SESSION "FKGoServer/FKServer_Game/Session"
)

//---------------------------------------------
const (
	DEFAULT_CH_IPC_SIZE = 16 // 默认玩家异步IPC消息队列大小
	CONST_ListenPort    = ":51000"
	SERVICE             = "[GAME]"
)

//---------------------------------------------
var (
	ERROR_INCORRECT_FRAME_TYPE = ERRORS.New("incorrect frame type")
	ERROR_SERVICE_NOT_BIND     = ERRORS.New("service not bind")
)

//---------------------------------------------
type Server struct{}

//---------------------------------------------
// 流消息接收器，进行消息接收
func (s *Server) func_Recv(stream PROTO.GameService_StreamServer, sess_die chan struct{}) chan *PROTO.Game_Frame {
	ch := make(chan *PROTO.Game_Frame, 1)
	go func() {
		defer func() {
			close(ch)
		}()
		for {
			in, err := stream.Recv()
			if err == IO.EOF { // 客户端已经关闭
				return
			}

			if err != nil {
				LOG.Error(err)
				return
			}
			select {
			case ch <- in:
			case <-sess_die:
			}
		}
	}()
	return ch
}

//---------------------------------------------
// 流解析处理器，核心逻辑
func (s *Server) Stream(stream PROTO.GameService_StreamServer) error {
	// 无论如何，最终要输出异常
	defer UTILS.PrintPanicStack()
	// 初始化会话
	var sess SESSION.Session
	sess_die := make(chan struct{})
	ch_agent := s.func_Recv(stream, sess_die)
	ch_ipc := make(chan *PROTO.Game_Frame, DEFAULT_CH_IPC_SIZE)

	// 无论如何，最终要关闭会话
	defer func() {
		LOGIC.Unregister(sess.UserId)
		close(sess_die)
		LOG.Debug("流关闭:", sess.UserId)
	}()

	// 从上下文中读取元数据
	md, ok := METADATA.FromContext(stream.Context())
	if !ok {
		LOG.Error("无法从上下文中读取元数据")
		return ERROR_INCORRECT_FRAME_TYPE
	}
	// 读取Key
	if len(md["userid"]) == 0 {
		LOG.Error("无法从元数据中获取UserID")
		return ERROR_INCORRECT_FRAME_TYPE
	}
	// 解析UserID
	userid, err := STRCONV.Atoi(md["userid"][0])
	if err != nil {
		LOG.Error(err)
		return ERROR_INCORRECT_FRAME_TYPE
	}

	// 进行用户注册
	sess.UserId = int32(userid)
	LOGIC.Register(sess.UserId, ch_ipc)
	LOG.Debug("UserID = ", sess.UserId, " 登陆")

	// 主消息循环
	for {
		select {
		case frame, ok := <-ch_agent: // 从Agent网关发来的frame消息
			if !ok { // 链接已经被关闭
				return nil
			}
			switch frame.Type {
			case PROTO.Game_Message: // 从 客户端->网关->游戏服务器 的消息
				// 获取其消息编号
				reader := PACKET.Reader(frame.Message)
				c, err := reader.ReadS16()
				if err != nil {
					LOG.Error(err)
					return err
				}
				// 查找消息处理函数
				handle := MSG.Handlers[c]
				if handle == nil {
					LOG.Error("该消息处理服务未被绑定:", c)
					return ERROR_SERVICE_NOT_BIND

				}

				// 处理请求消息
				ret := handle(&sess, reader)

				// 逻辑处理消息并构造Frame
				if ret != nil {
					if err := stream.Send(&PROTO.Game_Frame{Type: PROTO.Game_Message, Message: ret}); err != nil {
						LOG.Error(err)
						return err
					}
				}

				// 逻辑对会话的管理
				if sess.Flag&SESSION.SESS_KICKED_OUT != 0 { // 逻辑要求踢掉客户端
					if err := stream.Send(&PROTO.Game_Frame{Type: PROTO.Game_Kick}); err != nil {
						LOG.Error(err)
						return err
					}
					return nil
				}
			case PROTO.Game_Ping:
				if err := stream.Send(&PROTO.Game_Frame{Type: PROTO.Game_Ping, Message: frame.Message}); err != nil {
					LOG.Error(err)
					return err
				}
				LOG.Debug("处理ping消息")
			default:
				LOG.Error("incorrect frame type:", frame.Type)
				return ERROR_INCORRECT_FRAME_TYPE
			}
		case frame := <-ch_ipc: // 异步携程通讯消息
			if err := stream.Send(frame); err != nil {
				LOG.Error(err)
				return err
			}
		}
	}
}

//---------------------------------------------
