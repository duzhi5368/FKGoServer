//---------------------------------------------
package Service

//---------------------------------------------
import (
	LOG "log"
	OS "os"
	FILEPATH "path/filepath"
	STRINGS "strings"
	SYNC "sync"
	ATOMIC "sync/atomic"

	ETCD "github.com/coreos/etcd/client"
	CONTEXT "golang.org/x/net/context"
	GRPC "google.golang.org/grpc"
	CLI "gopkg.in/urfave/cli.v2"
)

//---------------------------------------------
// 一个客户端连接
type client struct {
	key  string
	conn *GRPC.ClientConn
}

// 一种类型的服务
type service struct {
	clients []client
	idx     uint32 // for round-robin purpose
}

// 全部服务类型
type service_pool struct {
	root           string
	names          map[string]bool
	services       map[string]*service
	names_provided bool
	client         ETCD.Client
	callbacks      map[string][]chan string
	mu             SYNC.RWMutex
}

var (
	_default_pool service_pool
	once          SYNC.Once
)

//---------------------------------------------
func InitWithCliContext(c *CLI.Context) {
	once.Do(func() { _default_pool.func_InitWithCliContext(c) })
}

//---------------------------------------------
func InitWithHostServices(root string, hosts, services []string) {
	once.Do(func() { _default_pool.func_initWithHostServices(root, hosts, services) })
}

//---------------------------------------------
func (p *service_pool) func_initWithHostServices(root string, hosts, services []string) {
	// init etcd client
	cfg := ETCD.Config{
		Endpoints: hosts, // c.StringSlice("etcd-hosts"),
		Transport: ETCD.DefaultTransport,
	}
	etcdcli, err := ETCD.New(cfg)
	if err != nil {
		LOG.Panic(err)
		OS.Exit(-1)
	}
	p.client = etcdcli
	p.root = root //c.String("etcd-root")

	// init
	p.services = make(map[string]*service)
	p.names = make(map[string]bool)

	// names init
	names := services // c.StringSlice("services")
	if len(names) > 0 {
		p.names_provided = true
	}

	LOG.Println("启动服务包括:", names)
	for _, v := range names {
		p.names[p.root+"/"+STRINGS.TrimSpace(v)] = true
	}

	// start connection
	p.connect_all(p.root)
}

//---------------------------------------------
func (p *service_pool) func_InitWithCliContext(c *CLI.Context) {
	// init etcd client
	cfg := ETCD.Config{
		Endpoints: c.StringSlice("etcd-hosts"),
		Transport: ETCD.DefaultTransport,
	}
	etcdcli, err := ETCD.New(cfg)
	if err != nil {
		LOG.Panic(err)
		OS.Exit(-1)
	}
	p.client = etcdcli
	p.root = c.String("etcd-root")

	// init
	p.services = make(map[string]*service)
	p.names = make(map[string]bool)

	// names init
	names := c.StringSlice("services")
	if len(names) > 0 {
		p.names_provided = true
	}

	LOG.Println("当前开启服务包括:", names)
	for _, v := range names {
		p.names[p.root+"/"+STRINGS.TrimSpace(v)] = true
	}

	// start connection
	p.connect_all(p.root)
}

//---------------------------------------------
// 连接全部服务
func (p *service_pool) connect_all(directory string) {
	kAPI := ETCD.NewKeysAPI(p.client)
	// get the keys under directory
	LOG.Println("连接服务目录:", directory)
	resp, err := kAPI.Get(CONTEXT.Background(), directory, &ETCD.GetOptions{Recursive: true})
	if err != nil {
		LOG.Println(err)
		return
	}

	// validation check
	if !resp.Node.Dir {
		LOG.Println("连接服务失败，这不是一个有效目录")
		return
	}

	for _, node := range resp.Node.Nodes {
		if node.Dir { // service directory
			for _, service := range node.Nodes {
				p.add_service(service.Key, service.Value)
			}
		}
	}
	LOG.Println("服务加载完毕")

	go p.watcher()
}

//---------------------------------------------
// 监视ETCD目录下的数据更变
func (p *service_pool) watcher() {
	kAPI := ETCD.NewKeysAPI(p.client)
	w := kAPI.Watcher(p.root, &ETCD.WatcherOptions{Recursive: true})
	for {
		resp, err := w.Next(CONTEXT.Background())
		if err != nil {
			LOG.Println(err)
			continue
		}
		if resp.Node.Dir {
			continue
		}

		switch resp.Action {
		case "set", "create", "update", "compareAndSwap":
			p.add_service(resp.Node.Key, resp.Node.Value)
		case "delete":
			p.remove_service(resp.PrevNode.Key)
		}
	}
}

//---------------------------------------------
// 增加一种服务
func (p *service_pool) add_service(key, value string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// name check
	service_name := FILEPATH.Dir(key)
	if p.names_provided && !p.names[service_name] {
		return
	}

	// try new service kind init
	if p.services[service_name] == nil {
		p.services[service_name] = &service{}
	}

	// create service connection
	service := p.services[service_name]
	if conn, err := GRPC.Dial(value, GRPC.WithBlock(), GRPC.WithInsecure()); err == nil {
		service.clients = append(service.clients, client{key, conn})
		LOG.Println("增加服务完成:", key, "-->", value)
		for k := range p.callbacks[service_name] {
			select {
			case p.callbacks[service_name][k] <- key:
			default:
			}
		}
	} else {
		LOG.Println("无法连接服务:", key, "-->", value, "错误信息:", err)
	}
}

//---------------------------------------------
// 移除一种服务
func (p *service_pool) remove_service(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// name check
	service_name := FILEPATH.Dir(key)
	if p.names_provided && !p.names[service_name] {
		return
	}

	// check service kind
	service := p.services[service_name]
	if service == nil {
		LOG.Println("移除服务失败，没有对应服务:", service_name)
		return
	}

	// remove a service
	for k := range service.clients {
		if service.clients[k].key == key { // deletion
			service.clients[k].conn.Close()
			service.clients = append(service.clients[:k], service.clients[k+1:]...)
			LOG.Println("服务移除完毕:", key)
			return
		}
	}
}

//---------------------------------------------
// 为一个服务提供一个ID
// 例如：path:/backends/snowflake, id:s1
// 于是调用该服务的路径则是 /backends/snowflake/s1
func (p *service_pool) get_service_with_id(path string, id string) *GRPC.ClientConn {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// check existence
	service := p.services[path]
	if service == nil {
		return nil
	}
	if len(service.clients) == 0 {
		return nil
	}

	// loop find a service with id
	fullpath := string(path) + "/" + id
	for k := range service.clients {
		if service.clients[k].key == fullpath {
			return service.clients[k].conn
		}
	}

	return nil
}

//---------------------------------------------
// get a service in round-robin style
// especially useful for load-balance with state-less services
func (p *service_pool) get_service(path string) (conn *GRPC.ClientConn, key string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// check existence
	service := p.services[path]
	if service == nil {
		return nil, ""
	}

	if len(service.clients) == 0 {
		return nil, ""
	}

	// get a service in round-robind style,
	idx := int(ATOMIC.AddUint32(&service.idx, 1)) % len(service.clients)
	return service.clients[idx].conn, service.clients[idx].key
}

//---------------------------------------------
func (p *service_pool) register_callback(path string, callback chan string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.callbacks == nil {
		p.callbacks = make(map[string][]chan string)
	}

	p.callbacks[path] = append(p.callbacks[path], callback)
	if s, ok := p.services[path]; ok {
		for k := range s.clients {
			callback <- s.clients[k].key
		}
	}
	LOG.Println("注册服务回调:", path)
}

//---------------------------------------------
func GetService(path string) *GRPC.ClientConn {
	conn, _ := _default_pool.get_service(_default_pool.root + "/" + path)
	return conn
}

//---------------------------------------------
func GetService2(path string) (*GRPC.ClientConn, string) {
	conn, key := _default_pool.get_service(_default_pool.root + "/" + path)
	return conn, key
}

//---------------------------------------------
func GetServiceWithId(path string, id string) *GRPC.ClientConn {
	return _default_pool.get_service_with_id(_default_pool.root+"/"+path, id)
}

//---------------------------------------------
func RegisterCallback(path string, callback chan string) {
	_default_pool.register_callback(_default_pool.root+"/"+path, callback)
}

//---------------------------------------------
