//---------------------------------------------
package main

//---------------------------------------------
import (
	FRAMEWORK "FKGoServer/FKGRpc_WordFilter/Framework"
	PROTO "FKGoServer/FKGRpc_WordFilter/Proto"
	NET "net"
	OS "os"

	LOG "github.com/Sirupsen/logrus"
	_ "FKGoServer/FKLib_Common/Profile"
	GRPC "google.golang.org/grpc"
)

//---------------------------------------------
const (
	_port = ":50002"
)

//---------------------------------------------
func main() {
	// 监听
	lis, err := NET.Listen("tcp", _port)
	if err != nil {
		LOG.Panic(err)
		OS.Exit(-1)
	}
	LOG.Info("listening on ", lis.Addr())

	// 注册服务
	s := GRPC.NewServer()
	ins := &FRAMEWORK.Server{}
	ins.Func_Init()
	PROTO.RegisterWordFilterServiceServer(s, ins)

	// 开始服务
	s.Serve(lis)
}

//---------------------------------------------
