//---------------------------------------------
package main

//---------------------------------------------
import (
	NET "net"
	OS "os"

	FRAMEWORK "FKGoServer/FKGRpc_Snowflake/Framework"
	PROTO "FKGoServer/FKGRpc_Snowflake/Proto"
	ETCDCLIENT "FKGoServer/FKLib_Common/ETCDClient"

	LOG "github.com/Sirupsen/logrus"
	_ "FKGoServer/FKLib_Common/Profile"
	GRPC "google.golang.org/grpc"
)

//---------------------------------------------
const (
	CONST_ListenPort = ":50003"
)

//---------------------------------------------
func main() {
	ETCDCLIENT.InitDefault()
	// 开启端口监听
	lis, err := NET.Listen("tcp", CONST_ListenPort)
	if err != nil {
		LOG.Panic(err)
		OS.Exit(-1)
	}
	LOG.Info("开启端口监听: ", lis.Addr())

	// 注册服务
	s := GRPC.NewServer() // 创建GRPC服务器
	ins := &FRAMEWORK.Server{}
	ins.Func_Init()
	PROTO.RegisterSnowflakeServiceServer(s, ins) // 注册服务

	// 开始服务，进行端口阻塞，等待进程被杀或者stop()函数被调用
	s.Serve(lis)
}

//---------------------------------------------
