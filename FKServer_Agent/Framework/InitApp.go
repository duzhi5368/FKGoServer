//---------------------------------------------
package framework

//---------------------------------------------
import (
	BINARY "encoding/binary"
	IO "io"
	NET "net"
	OS "os"
	TIME "time"

	SERVICES "FKGoServer/FKLib_Common/Service"
	UTILS "FKGoServer/FKLib_Common/Utils"
	SESSION "FKGoServer/FKServer_Agent/Session"

	LOG "github.com/Sirupsen/logrus"
	KCP "github.com/xtaci/kcp-go"
	CLI "gopkg.in/urfave/cli.v2"
)

//---------------------------------------------
// 启动
func Func_InitApp(c *CLI.Context) {
	// 启动新协程处理Unix内部信号
	go func_HandlerUnixSign()
	// 服务实际初始化
	SERVICES.InitWithCliContext(c)
}

//---------------------------------------------
// 新线程，开启TCP服务器监听
func Func_StartTcpServer() {
	// 获取一个TCP地址
	tcpAddr, err := NET.ResolveTCPAddr("tcp4", CONST_ListerPort)
	func_CheckError(err)
	// 创建一个该地址的监听
	listener, err := NET.ListenTCP("tcp", tcpAddr)
	func_CheckError(err)

	LOG.Info("正在监听TCP地址:", listener.Addr())

	// 死循环接收连接
	for {
		// 始终在accept等待
		conn, err := listener.AcceptTCP()
		if err != nil {
			LOG.Warning("接收客户端连接失败:", err)
			continue
		}
		// 设置Socket读取缓冲
		conn.SetReadBuffer(CONST_SendBuffer)
		// 设置Socket发送缓冲
		conn.SetWriteBuffer(CONST_ReceiveBuffer)
		// 开启新协程处理这次连接接收的数据
		go func_HandleNewClientConnect(conn)

		// 收到服务器的死亡消息
		select {
		case <-DIE_SIGN:
			listener.Close() // 关闭监听
			return           // 退出循环
		default:
		}
	}
}

//---------------------------------------------
// 新线程，开启UDP服务器监听
func Func_StartUdpServer() {
	// 直接监听一个端口
	l, err := KCP.Listen(CONST_ListerPort)
	func_CheckError(err)

	LOG.Info("正在监听UDP地址:", l.Addr())

	// 获取连接对象
	lis := l.(*KCP.Listener)
	// 设置Socket读取缓冲
	if err := lis.SetReadBuffer(CONST_UdpBuffer); err != nil {
		LOG.Println(err)
	}
	// 设置Socket发送缓冲
	if err := lis.SetWriteBuffer(CONST_UdpBuffer); err != nil {
		LOG.Println(err)
	}
	if err := lis.SetDSCP(CONST_TosEF); err != nil {
		LOG.Println(err)
	}

	// 死循环接收连接
	for {
		// 始终在accept等待
		conn, err := lis.AcceptKCP()
		if err != nil {
			LOG.Warning("接收客户端连接失败:", err)
			continue
		}
		// 设置KCP参数
		conn.SetWindowSize(32, 32)
		conn.SetNoDelay(1, 20, 1, 1)
		conn.SetKeepAlive(0)
		conn.SetStreamMode(true)

		// 开启新协程处理这次连接接收的数据
		go func_HandleNewClientConnect(conn)

		// 收到服务器的死亡消息
		select {
		case <-DIE_SIGN:
			lis.Close() // 关闭监听
			return      // 退出循环
		default:
		}
	}
}

//---------------------------------------------
// 处理客户端连接消息
// 这个函数是在单独一个协程中执行的，进行接入包解析
// 每个消息包格式定义如下：头两个字节为DATA数据大小
// | 2B size |     DATA       |
//
func func_HandleNewClientConnect(conn NET.Conn) {
	// 无论如何，最后退出时总要打印产生panic时的调用栈
	defer UTILS.Func_PrintPanicStack()

	// 头两个字节缓冲区
	header := make([]byte, 2)
	// 输入Channel
	in := make(chan []byte)
	defer func() {
		close(in) // 无论如何，最终退出时总是要关闭Session连接的
	}()

	// 为连接创建一个新的Session对象，并记录其IP
	var sess SESSION.Session
	host, port, err := NET.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		LOG.Error("无法获得远程连接地址:", err)
		return
	}
	sess.IP = NET.ParseIP(host)
	LOG.Infof("接收到远程连接地址:%v 端口:%v", host, port)

	// 对话死亡消息
	sess.Die = make(chan struct{})

	// 创建一个写入缓冲区
	out := func_CreateWriteBuffer(conn, sess.Die)
	// 创建新协程进行包发送
	go out.func_StartSendPacket()

	// 增加一个引用
	SYNC_WAITGROUP.Add(1)
	// 为这个会话启动一个协程进行事务处理
	go func_ClientSessionHandler(&sess, in, out)

	// 死循环接收连接
	for {
		// 如果客户端和服务器之间的物理通讯出现故障，将导致读取时出现持续Block
		// 所以这里增加TimeOut用来解决类似的死链接
		conn.SetReadDeadline(TIME.Now().Add(CONST_ReadDeadline * TIME.Second))

		// 读取两字节头
		n, err := IO.ReadFull(conn, header)
		if err != nil {
			LOG.Warningf("读取两字节头失败,会话IP:%v 错误原因:%v 实际数据包大小:%v", sess.IP, err, n)
			return
		}
		size := BINARY.BigEndian.Uint16(header)

		// 分配一块内存用来进行数据读取
		payload := make([]byte, size)
		n, err = IO.ReadFull(conn, payload)
		if err != nil {
			LOG.Warningf("读取数据失败,会话IP:%v 错误原因:%v 预期数据包大小:%v", sess.IP, err, n)
			return
		}

		// 将收到的数据进行压栈
		select {
		case in <- payload: // 将读取到的数据进行压栈
		case <-sess.Die: // 收到关闭死亡消息
			LOG.Warningf("逻辑发布死亡消息,连接被关闭,会话标示:%v 会话IP:%v", sess.Flag, sess.IP)
			return
		}
	}
}

//---------------------------------------------
// 对严重错误进行处理
func func_CheckError(err error) {
	if err != nil {
		LOG.Fatal(err)
		OS.Exit(-1)
	}
}

//---------------------------------------------
