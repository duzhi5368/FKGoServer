//---------------------------------------------
package main

//---------------------------------------------
import (
	NET "net"
	HTTP "net/http"
	OS "os"
	TIME "time"

	DB "FKGoServer/FKLib_Common/DB"
	ETCDCLIENT "FKGoServer/FKLib_Common/ETCDClient"
	SERVICE "FKGoServer/FKLib_Common/Service"
	NUMBERS "FKGoServer/FKLib_Common/Utils"
	FRAMEWORK "FKGoServer/FKServer_Game/Framework"
	PROTO "FKGoServer/FKServer_Game/Proto"

	LOG "github.com/Sirupsen/logrus"
	GRPC "google.golang.org/grpc"
	CLI "gopkg.in/urfave/cli.v2"
)

//---------------------------------------------
func main() {
	// 开启新协程进行端口监听
	go HTTP.ListenAndServe("0.0.0.0:6060", nil)

	app := &CLI.App{
		Name: "agent",
		Flags: []CLI.Flag{
			&CLI.StringFlag{
				Name:  "listen",
				Value: ":8888",
				Usage: "监听地址：端口",
			},
			&CLI.StringSliceFlag{
				Name:  "etcd-hosts",
				Value: CLI.NewStringSlice("http://127.0.0.1:2379"),
				Usage: "etcd主机",
			},
			&CLI.StringFlag{
				Name:  "etcd-root",
				Value: "/backends",
				Usage: "etcd根目录",
			},
			&CLI.StringFlag{
				Name:  "numbers",
				Value: "/numbers",
				Usage: "etcd中的Number目录",
			},
			&CLI.StringSliceFlag{
				Name:  "services",
				Value: CLI.NewStringSlice("snowflake-10000"),
				Usage: "自动发现服务器",
			},
			&CLI.StringFlag{
				Name:  "mongodb",
				Value: "mongodb://127.0.0.1/mydb",
				Usage: "mongodb路径",
			},
			&CLI.DurationFlag{
				Name:  "mongodb-timeout",
				Value: 30 * TIME.Second,
				Usage: "mongodb连接超时时间",
			},
			&CLI.IntFlag{
				Name:  "mongodb-concurrent",
				Value: 128,
				Usage: "mongodb并发查询数",
			},
		},
		Action: func(c *CLI.Context) error {
			LOG.Println("监听端口:", c.String("listen"))
			LOG.Println("etcd主机:", c.StringSlice("etcd-hosts"))
			LOG.Println("etcd根目录:", c.String("etcd-root"))
			LOG.Println("启动服务:", c.StringSlice("services"))
			LOG.Println("Numbers目录:", c.String("numbers"))
			LOG.Println("mongodb地址:", c.String("mongodb"))
			LOG.Println("mongodb连接超时时间:", c.Duration("mongodb-timeout"))
			LOG.Println("mongodb最大并发查询数:", c.Int("mongodb-concurrent"))

			// 监听
			lis, err := NET.Listen("tcp", FRAMEWORK.CONST_ListenPort)
			if err != nil {
				LOG.Panic(err)
				OS.Exit(-1)
			}
			LOG.Info("正在监听 ", lis.Addr())

			// 注册服务
			s := GRPC.NewServer()
			ins := new(FRAMEWORK.Server)
			PROTO.RegisterGameServiceServer(s, ins)

			// 初始化Services
			ETCDCLIENT.Init(c.StringSlice("etcd-hosts"))
			SERVICE.InitWithHostServices(c.String("etcd-root"), c.StringSlice("etcd-hosts"), c.StringSlice("services"))
			NUMBERS.Fun_Init(c.String("numbers"))
			DB.Func_InitDB(c.String("mongodb"), c.Int("mongodb-concurrent"), c.Duration("mongodb-concurrent"))

			// 开始服务
			return s.Serve(lis)
		},
	}
	app.Run(OS.Args)
}

//---------------------------------------------
