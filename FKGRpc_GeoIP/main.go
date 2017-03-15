//---------------------------------------------
package main

//---------------------------------------------
import (
	PROTO "FKGoServer/FKGRpc_GeoIP/Proto"
	NET "net"
	OS "os"

	FRAMEWORK "FKGoServer/FKGRpc_GeoIP/Framework"
	LOG "github.com/Sirupsen/logrus"
	GRPC "google.golang.org/grpc"
)

//---------------------------------------------
const (
	_port = ":50000"
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
	PROTO.RegisterGeoIPServiceServer(s, ins)

	// 开始服务
	s.Serve(lis)
}

//---------------------------------------------
