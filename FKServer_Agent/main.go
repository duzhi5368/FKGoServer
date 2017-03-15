//---------------------------------------------
package main

//---------------------------------------------
/*
	请参见Doc/ReadMe.txt
*/
//---------------------------------------------
import (
	_ "net/http/pprof"

	HTTP "net/http"
	OS "os"

	UTILS "FKGoServer/FKLib_Common/Utils"
	FRAMEWORK "FKGoServer/FKServer_Agent/Framework"

	LOG "github.com/Sirupsen/logrus"
	CLI "gopkg.in/urfave/cli.v2"
)

//---------------------------------------------
// main入口函数
func main() {
	// 设置日志层级
	LOG.SetLevel(LOG.DebugLevel)

	// 无论如何，在最终退出之前，捕获全部异常堆并打印出来
	defer UTILS.Func_PrintPanicStack()

	// 创建监听协程，监听HTTP 6060端口
	go HTTP.ListenAndServe("0.0.0.0:6060", nil)

	// 添加命令行参数
	app := &CLI.App{
		Name: "FKAgent",
		Flags: []CLI.Flag{
			&CLI.StringFlag{
				Name:  "listen",
				Value: ":8888",
				Usage: "监听端口",
			},
			&CLI.StringSliceFlag{
				Name:  "etcd-hosts",
				Value: CLI.NewStringSlice("http://127.0.0.1:2379"),
				Usage: "etcd服务器地址",
			},
			&CLI.StringFlag{
				Name:  "etcd-root",
				Value: "/backends",
				Usage: "etcd根目录",
			},
			&CLI.StringSliceFlag{
				Name:  "services",
				Value: CLI.NewStringSlice("snowflake-10000", "game-10000"),
				Usage: "自动发现服务器",
			},
		},
		Action: func(c *CLI.Context) error {
			LOG.Println("监听端口:", c.String("listen"))
			LOG.Println("etcd服务器地址:", c.StringSlice("etcd-hosts"))
			LOG.Println("etcd根目录:", c.String("etcd-root"))
			LOG.Println("自动发现依赖服务:", c.StringSlice("services"))

			// 初始化服务
			FRAMEWORK.Func_InitApp(c)

			// 启动TCP和UDP服务器监听
			go FRAMEWORK.Func_StartTcpServer()
			go FRAMEWORK.Func_StartUdpServer()

			// 持续等待
			select {}
		},
	}

	// Cli执行函数
	app.Run(OS.Args)
}

//---------------------------------------------
