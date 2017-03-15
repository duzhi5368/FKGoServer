//---------------------------------------------
package ETCDClient

//---------------------------------------------
import (
	OS "os"
	STRINGS "strings"

	LOG "github.com/Sirupsen/logrus"
	ETCD "github.com/coreos/etcd/client"
)

//---------------------------------------------
const (
	DEFAULT_ETCD = "http://172.17.42.1:2379"
)

//---------------------------------------------
var client ETCD.Client

//---------------------------------------------
func InitDefault() {
	var machines []string
	// etcd client
	machines = []string{DEFAULT_ETCD}
	if env := OS.Getenv("ETCD_HOST"); env != "" {
		machines = STRINGS.Split(env, ";")
	}

	// ETCD客户端配置
	cfg := ETCD.Config{
		Endpoints: machines,
		Transport: ETCD.DefaultTransport,
	}

	// 创建ETCD客户端
	c, err := ETCD.New(cfg)
	if err != nil {
		LOG.Error(err)
		return
	}
	client = c
}

//---------------------------------------------
func Init(host []string) {
	// 设置配置
	cfg := ETCD.Config{
		Endpoints: host,
		Transport: ETCD.DefaultTransport,
	}

	// 创建客户端
	etcdcli, err := ETCD.New(cfg)
	if err != nil {
		LOG.Panic(err)
		return
	}
	client = etcdcli
}

//---------------------------------------------
func KeysAPI() ETCD.KeysAPI {
	return ETCD.NewKeysAPI(client)
}

//---------------------------------------------
func NewOptions() ETCD.GetOptions {
	return ETCD.GetOptions{}
}

//---------------------------------------------
func NewWatcherOptions(recursive bool) *ETCD.WatcherOptions {
	return &ETCD.WatcherOptions{Recursive: recursive}
}

//---------------------------------------------
