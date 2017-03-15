//---------------------------------------------
package main
//---------------------------------------------
import (
	NET "net"
	HTTP "net/http"
	OS "os"
	TIME "time"

	CLI "gopkg.in/urfave/cli.v2"
	LOG "github.com/Sirupsen/logrus"
	LOGRUSHHOOKS "github.com/xtaci/logrushooks"
	GRPC "google.golang.org/grpc"

	KAFKA "FKGoServer/FKGRpc_Chat/Kafka"
	FRAMEWORK "FKGoServer/FKGRpc_Chat/Framework"
	PROTO "FKGoServer/FKGRpc_Chat/Proto"
)
//---------------------------------------------
func main() {
	LOG.AddHook(LOGRUSHHOOKS.LineNoHook{})

	go func() {
		LOG.Info(HTTP.ListenAndServe("0.0.0.0:6060", nil))
	}()
	app := &CLI.App{
		Name: "chat",
		Flags: []CLI.Flag{
			&CLI.StringFlag{
				Name:  "listen",
				Value: ":10000",
				Usage: "listening address:port",
			},
			&CLI.StringFlag{
				Name:  "kafka-bucket",
				Value: "kafka-bucket",
				Usage: "key with kafka offset",
			},
			&CLI.StringFlag{
				Name:  "chat-topic",
				Value: "chat_updates",
				Usage: "chat topic in kafka",
			},
			&CLI.StringSliceFlag{
				Name:  "kafka-brokers",
				Value: CLI.NewStringSlice("127.0.0.1:9092"),
				Usage: "kafka brokers address",
			},
			&CLI.StringFlag{
				Name:  "boltdb",
				Value: "/data/CHAT.DAT",
				Usage: "chat snapshot file",
			},
			&CLI.StringFlag{
				Name:  "bucket",
				Value: "EPS",
				Usage: "bucket name",
			},
			&CLI.IntFlag{
				Name:  "retention",
				Value: 1024,
				Usage: "retention number of messags for each endpoints",
			},
			&CLI.DurationFlag{
				Name:  "write-interval",
				Value: 10 * TIME.Minute,
				Usage: "chat message persistence interval",
			},
		},

		Action: func(c *CLI.Context) error {
			LOG.Println("listen:", c.String("listen"))
			LOG.Println("boltdb:", c.String("boltdb"))
			LOG.Println("kafka-brokers:", c.StringSlice("kafka-brokers"))
			LOG.Println("chat-topic", c.String("chat-topic"))
			LOG.Println("bucket:", c.String("bucket"))
			LOG.Println("retention:", c.Int("retention"))
			LOG.Println("write-interval:", c.Duration("write-interval"))
			LOG.Println("kafka-bucket", c.String("kafka-bucket"))
			LOG.Println("kafka-brokers", c.StringSlice("kafka-brokers"))
			// 监听
			lis, err := NET.Listen("tcp", c.String("listen"))
			if err != nil {
				LOG.Panic(err)
				OS.Exit(-1)
			}
			LOG.Info("listening on:", lis.Addr())

			KAFKA.Init(c)
			// 注册服务
			s := GRPC.NewServer()
			ins := &FRAMEWORK.Server{}
			ins.Func_Init(c)
			PROTO.RegisterChatServiceServer(s, ins)
			// 开始服务
			return s.Serve(lis)
		},
	}
	app.Run(OS.Args)
}
//---------------------------------------------