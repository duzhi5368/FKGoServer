### TodoList
- [x] 2016/8/26   设计好服务器目录层级
- [x] 2016/8/26   添加完成依赖库以及Vendor.json依赖库配置
- [x] 2016/9/12   整理完成FKCommmonLib
- [x] 2016/12/2   整理完成Agent, Game
- [x] 2017/1/2    整理完成SnowFlake, WordFilter, GeoIP
- [x] 2017/2/3    整理完成FKCuiClient
- [x] 2017/2/9    整理完成Tools
- [x] 2017/3/14   整理完成Chat
- [x] 2017/3/14   去除不受控的Vendor库
- [x] 2017/3/16   整理完成Doc文档
- [ ] 添加大量单元test测试
- [ ] 提交环境部署，并进行整合测试
- [ ] 去冗余重复功能库，小单元依赖库整合
- [ ] 向其他GameServer进行学习调整

:brush::brush::brush::brush::brush:

# 说明

[FKGoServer](https://github.com/duzhi5368)是一套游戏服务器框架。
* 它采用[Go语言](http://golang.org)开发
* 它采用[HTTP/2](http://http2.github.io/)作为服务器组内部主要通讯协议
* 它采用[ProtoBuf](https://github.com/google/protobuf)作为服务间数据通讯格式
* 它采用[MicroService](http://martinfowler.com/articles/microservices.html)作为架构思想
* 它采用[Docker](https://www.docker.com/)作为服务发布手段

> 开发工具：  *IntelliJ IDEA ULTIMATE 2016*

> 环境依赖：  *Go 1.7.1*

## 基本项目组成

* **FKLib_Common**          一些基本常用功能库包：提供给FKServer系列使用
* **FKServer_Agent**        网关服务器，负责客户端连接安全以及消息解密转发
* **FKServer_Game**         游戏逻辑服务器，负责客户端主体逻辑，它以RPC方式调用其他负责处理具体业务
* **FKGRpc_Chat**           微服务：聊天功能
* **FKGRpc_GeoIP**          微服务：查询用户IP所属国，省，地区功能
* **FKGRpc_Rank**           微服务：排名功能
* **FKGRpc_Snowflake**      微服务：生成唯一UUID
* **FKGRpc_WordFilter**     微服务：脏字敏感词过滤功能
* **FKTools_Dsicover**      工具：进行微服务测试
* **FKTools_GenApi**        工具：生成客户端协议代码
* **FKTools_GenProto**      工具：生成服务器协议代码
* **FKTools_Simulate**      工具：消息模拟器
* **FKTools_CuiClient**     工具：Kafka可视化客户端

# 项目组成部分说明

## 1. Agent Server - 网关服务器

### 用途

Go语言编写的，用于游戏服务器组的，负责网关功能的服务器。

### 特性

* 处理各种协议的接入，同时支持 TCP 和 UDP (KCP协议)，进行双栈通信。
* 连接管理，会话建立，数据包加解密(DH+RC4)。
* **透传**解密后的原始数据流到后端（通过gRPC streaming)。
* **复用**多路用户连接，到一条通往游戏服务器的物理连接。
* 可以不断开连接切换后端业务。
* 提供唯一入口，安全隔离核心服务。

### 协议号划分

数据包会根据协议编号（0-65535) **透传** 到对应的服务， 例如(示范）:      

      1-1000: 登陆相关协议，网关协同auth服务处理。
      1001-10000: 游戏逻辑段
      ....
      
具体的划分根据业务需求进行扩展或调整。

### 消息封包格式
 
        +----------------------------------------------------------------+     
        | SIZE(2) | PACKINDEX(4) | PROTO(2) | PAYLOAD(SIZE-6)            |     
        +----------------------------------------------------------------+     
             
        SIZE: 后续数据包总长度         
        PACKINDEX: 数据包序号           
        PROTO: 协议号           
        PAYLOAD: 实际负载           
### 安装
参考Dockerfile

## 2. Game Server - 逻辑服务器

### 设计理念
游戏服务器对Agent只提供一个接口， 即:
        
        rpc Stream(stream Game.Frame) returns (stream Game.Frame);
        
该接口用来接收来自Agent的请求Frame流，并返回给Agent对应的响应Frame流。
而来自Agent的Frame大体分为两类：  
* 流程控制类（register, kick)     
* 来自客户端的，经过agent解密后的数据包 (message)       

数据包(message)格式为:      

    协议号＋数据
    +----------------------------------+     
    | PROTO(2) | PAYLOAD(n)            |     
    +----------------------------------+     

在**Msg**目录中绑定对应函数进行处理，协议生成和绑定通过**FKTools_GenApi**和**FKTools_GenProto**进行。

### 安装
参考Dockerfile


## 3. RPC_SnowFlake 唯一ID生成器

### 用途
在分布式系统中，生成大量的唯一的有序的ID。它是一个分布式uuid发生器，Twitter snowflake的go语言版本。 
### 设计理念
uuid格式为:

    +-------------------------------------------------------------------------------------------------+
    | UNUSED(1BIT) |         TIMESTAMP(41BIT)           |  MACHINE-ID(10BIT)  |   SERIAL-NO(12BIT)    |
    +-------------------------------------------------------------------------------------------------+
    UNUSED:         1bit    不使用，永远为0
    TIMESTAMP:      41bit   时间戳，毫秒级时间（41位可支持使用69.7年）
    MACHINE-ID:     10bit   工作机器ID（10位长度，允许部署1023个分布节点。该值可以使用机器MAC地址或IP+Path）
    SERIAL-NO:      12bit   序列号（12位计数支持每个节点每毫秒生成4095个ID序列号,这些ID进行自增即可，若1毫秒使用了4095以上的序列号，需等待至下一毫秒）

### 安装 
默认情况下uuid发生器依赖的snowflake-uuid键值对必须预先在etcd中创建，snowflake启动的时候会读取，例如： 

       curl http://172.17.42.1:2379/v2/keys/seqs/snowflake-uuid -XPUT -d value="0"          

这个snowflake-uuid会用于MACHINE-ID的自动生成，如果完全由用户自定义machine_id，可以通过环境变量指定，如:

       export MACHINE_ID=123

如果要使用序列发生器Next()，必须预先创建一个key，例如:       

       curl http://172.17.42.1:2379/v2/keys/seqs/userid -XPUT -d value="0"          
        

### 使用
参考测试用例 `snowflake.proto` 文件

### 环境变量
    ETCD_HOST:    eg: http://172.17.42.1:2379       
    MACHINE_ID:   eg: 123
    

## 4. RPC_WordFilter - 文字过滤

### 设计理念
基于[Sego](https://github.com/huichen/sego) 实现

1. 首先对文本进行分词
1. 然后和脏词库中的词汇进行比对，时间复杂度为O(m)， 其中m为需要处理的消息长度, 和脏词库的大小无关。              

### 使用
参考测试用例 `wordfilter.proto` 文件

### 安装部署
基于分词的文字过滤会消耗大量内存，**RPC_WordFilter**至少需要500M内存才能运行，建议每实例配置1GB。

## 5. RPC_GeoIP - IP查询

### 设计思路
查询IP归属地，基于maxmind的geoip2库做的封装，如果需要最新的准确的数据，需要向maxmind购买。     
(query geo-locations of IP, if you need accurate & updated data, please purchase from maxmind.com, thanks. )        

> 问: 为什么选择maxmind的geoip2库？         
> 答: maxmind的geoip2的库设计为一个支持mmap的二叉树文件，查询时间复杂度为O(logN),        文件大小不超过100M，极其紧凑，省内存，速度快，零配置，是目前见过的最好的方案。                

### 使用
参考测试用例

### 安装
参考Dockerfile

##  6. RPC_Chat - 聊天

### 设计理念

     EndPoint:  消息收发点（对应到一个玩家的聊天，或者一个联盟聊天） 
     PubSub: 对任意EndPoint进行发布，订阅

* 聊天服务器并不关心一个EndPoint对应的是一个玩家，还是一个联盟，只需要一个独立的id(snowflake-id)。       
* 通常在玩家注册的时候，会创建一个EndPoint，创建一个联盟的时候，也会创建一个EndPoint。     
* 玩家登陆后，会订阅到自己的私人EndPoint和所属联盟的EndPoint，以便接受实时聊天消息。      
* CHAT会保留一定数量的消息在内存中（默认128条），这个消息队列会定期持久化到本地磁盘，以便重启时候加载。       
* 持久化采用boltdb，零配置, 数据存储在 VOLUME /data。        

基于PubSub的聊天服务器，优点在于可以通过多个途径**同时**访问到同一个EndPoint, 例如：      
1. 游戏内     
2. 游戏提供的离线聊天工具     
3. 提供给XMPP网关       

### 使用
参考测试用例以及chat.proto文件

### 安装
参考Dockerfile

##  7. RPC_Rank - 排名

### 设计理念
* 对int32类型的id, score进行排名， 并用boltdb实现持久化。      
* 排名依据score进行，可以获得范围，比如［1，100］名的列表，可以定位某个玩家的排名，比如id为1234的排名。      
* 排名包含无限个集合，根据id(snowflake-id)区分，用户根据业务需求创建。          
* 持久化采用boltdb，零配置, 数据存储在 volume /data 。      

### 性能
采用混合策略做排名:           
1. 数据量小于1024(暂定)的时候，采用sortedset实现, 大部分操作时间复杂度为O(n)       
2. 超过之后采用rbtree实现, 时间复杂度O(logN)。         

sortedset的紧凑存储结构能充分利用cpu cache，而对rbtree的访问基本是全部cache miss; 所以必须在达到一定数据量之后，算法时间复杂度提升才能弥补cache miss.         

### 使用
参考测试用例以及rank.proto文件

### 安装
参考Dockerfile

# 工具说明

    TODO:

# 支持库说明

| 链接  | 说明 |
| ----- | ----------- |
|  **github/Shopify/sarama**                        |  是一个针对Apache Kafka的客户端库。|  
|  **github.com/Sirupsen/logrus**                   |  是一个和标准库中的logger完全兼容的加强日志库。|  
|  **github.com/adamzy/cedar-go**                   |  是一个非线程安全的DAT（双数组字典树）的实现。|  
|  **github.com/boltdb/bolt**                       |  是一个小型的嵌入式的，可序列化的，事务性的，Key/Value存储数据库。|  
|  **github.com/coreos/etcd**                       |  是一个分布式的Key/Value存储软件，用于配置共享和服务发现。|  
|  **github.com/davecgh/go-spew/spew**              |  是一个调试分析时常使用的，对Go数据进行格式化输出的库。|  
|  **github.com/eapache/go-resiliency/breaker** |  是一个简单的熔断器模式。|  
|  **github.com/eapache/queue**                     |  是一个线程不安全的，但内存占用和时间效率很高的环形缓冲区队列。|  
|  **github.com/golang/protobuf/proto**             |  是一个Google Protobuf的Go语言版本。|  
|  **github.com/golang/snappy**                     |  是一个Snappy压缩算法的Go语言版本，该算法的压缩速率极快。|  
|  **github.com/huichen/sego**                      |  是一个进行中文分词的库。|  
|  **github.com/jroimartin/gocui**                  |  是一个创建控制台的用户界面库。|  
|  **github.com/peterbourgon/g2s**                  |  是一个简单的对StatsD的封装库，是用来性能监察的。|  
|  **github.com/oschwald/maxminddb-golang**         |  是一个MaxMind DB格式的读取器库。|  
|  **github.com/tealeg/xlsx**                       |  是一个读取和写入Micrsoft Excel生成的xlsx格式的库。|  
|  **github.com/klauspost/cpuid**                   |  是一个获取本机CPU信息的库。|  
|  **github.com/klauspost/crc32**                   |  是一个超快性能的Crc32哈希库。|  
|  **github.com/klauspost/reedsolomon**             |  是一个超快性能的里德-所罗门纠错算法。|  
|  **github.com/mattn/go-runewidth**                |  是一个获取字符或字符串宽度的函数库，支持多语言。|  
|  **github.com/nsf/termbox-go**                    |  是一个简易的，跨平台的，基于文本操作的用户界面。|  
|  **github.com/pkg/errors**                        |  是一个错误处理的加强包。|  
|  **github.com/pierrec/lz4**                       |  是一个LZ4压缩和解压缩库。|  
|  **github.com/pierrec/xxHash/xxHash32**       |  是一个非常快速的散列算法库。|  
|  **github.com/rcrowley/go-metrics**           |  是一个类似Java Metrics的便捷的程序性能监测库。|  
|  **github.com/ugorji/go/codec**               |  是一个高性能的，功能丰富的常用编码/解码库和RPC库。|  
|  **github.com/xtaci/logrushooks**             |  是一个简单的Logrus的Hook库。|  
|  **github.com/xtaci/kcp-go**                  |  是一个快速的，可靠的，有序的UDP库。|  
|  **gopkg.in/mgo.v2**                          |   是一个MangoDB的Go语言驱动库。|  
|  **gopkg.in/urfave/cli.v2**                   |  是一个构建控制台命令行的库。|  
|  **gopkg.in/vmihailenco/msgpack.v2**          |  是一个高效的二进制序列化格式，类似Protobuf,但性能更好。|  
|  **google.golang.org/grpc**                   |  是一个高性能的远程过程调用(RPC)框架，面向移动和HTTP/2设计。|  
|  **github.com/golang/x/net/http2**            |  是一个google自带的HTTP2通讯协议库。|  

### github/Shopify/sarama

**用途**

Sarama是一个GO语言开发的，针对[Apache Kafka](https://kafka.apache.org/) 0.8以上版本的客户端库。

**补充说明**
* 尽量使用Kafka 0.10.1 以上版本
* 如果想使用Kafka Consumer group(消费者组)，那么
    * 针对Kafka0.8.2版本和更早版本，可使用基于Zookeeper的跟踪 https://github.com/wvanbergen/kafka
    * 针对Kafka0.9以及更高版本，可使用 https://github.com/bsm/sarama-cluster
* Sarama通过gopkg.in服务来保证API的稳定性，所以如果你需要稳定的API版本，应当从http://gopkg.in/Shopify/sarama.v1获取

**基本流程**

* [Go Doc](https://godoc.org/github.com/Shopify/sarama)

**推荐资料**
    
* [官网](https://github/Shopify/sarama)
* [Wiki](https://github/Shopify/sarama/wiki)
* [The Guide of Kafka](https://cwiki.apache.org/confluence/display/KAFKA/A+Guide+To+The+Kafka+Protocol)
* [Kafka论坛](https://groups.google.com/forum/#!forum/kafka-clients)
    
### github.com/Sirupsen/logrus

**用途**

logrus是一个结构化的Go日志库，它和标准库中的logger完全兼容。

**补充说明**
* 因为它完全兼容标准库stdlib中的logger，所以你可以`log` 直接替换为 `log "github.com/Sirupsen/logrus"` 即可。
* 使用 `log.SetFormatter(&log.JSONFormatter{})` 可以很容易被 logstask 或者 Splunk 解析。

```json
{"animal":"walrus","level":"info","msg":"A group of walrus emerges from the ocean",
"size":10,"time":"2014-03-10 19:57:38.562264131 -0400 EDT"}

{"level":"warning","msg":"The group's number increased tremendously!",
"number":122,"omg":true,"time":"2014-03-10 19:57:38.562471297 -0400 EDT"}

{"level":"fatal","msg":"The ice breaks!","number":100,"omg":true,
"time":"2014-03-10 19:57:38.562543128 -0400 EDT"}
```

* 使用 `log.SetFormatter(&log.JSONFormatter{})` 默认格式，则非常可以被 [logfmt](http://godoc.org/github.com/kr/logfmt) 兼容。
```text
time="2015-03-26T01:27:38-04:00" level=debug msg="Started observing beach" animal=walrus number=8
time="2015-03-26T01:27:38-04:00" level=info msg="A group of walrus emerges from the ocean" animal=walrus size=10
time="2015-03-26T01:27:38-04:00" level=warning msg="The group's number increased tremendously!" number=122 omg=true
time="2015-03-26T01:27:38-04:00" level=debug msg="Temperature changes" temperature=-4
time="2015-03-26T01:27:38-04:00" level=panic msg="It's over 9000!" animal=orca size=9009
time="2015-03-26T01:27:38-04:00" level=fatal msg="The ice breaks!" err=&{0x2082280c0 map[animal:orca size:9009] 2015-03-26 01:27:38.441574009 -0400 EDT panic It's over 9000!} number=100 omg=true
```
* 我们更建议日志的可读性，尽量使用
```go
log.WithFields(log.Fields{
  "event": event,
  "topic": topic,
  "key": key,
}).Fatal("Failed to send event")
```
去替代
```go
log.Fatalf("Failed to send event %s to topic %s with key %d")
```
* 支持进行日志级别的Hook。例如将一些“错误”以上级别的日志发送给自己的处理系统，例如
```go
import (
  log "github.com/Sirupsen/logrus"
  "gopkg.in/gemnasium/logrus-airbrake-hook.v2" // 一个日志Hook系统
  logrus_syslog "github.com/Sirupsen/logrus/hooks/syslog"
  "log/syslog"
)

func init() {

  // 创建自定义的Hook
  log.AddHook(airbrake.NewHook(123, "xyz", "production"))

  hook, err := logrus_syslog.NewSyslogHook("udp", "localhost:514", syslog.LOG_INFO, "")
  if err != nil {
    log.Error("Unable to connect to local syslog daemon")
  } else {
    log.AddHook(hook)
  }
}
```

* 错误等级
```go
log.Debug("Useful debugging information.")
log.Info("Something noteworthy happened!")
log.Warn("You should probably take a look at this.")
log.Error("Something failed but I'm not quitting.")
// 日志写入后会调用 os.Exit(1)
log.Fatal("Bye.")
// 日志写入后会调用 panic()
log.Panic("I'm bailing.")
```
你可以指定写入日志的日志等级
```go
// 这将会写入指定级别的日志，以及高于该级别的日志
log.SetLevel(log.InfoLevel)
```
* 第三方日志格式化工具:

    * [`logstash`](https://github.com/bshuster-repo/logrus-logstash-hook). 将日志格式化为[Logstash](http://logstash.net)格式.
    * [`prefixed`](https://github.com/x-cray/logrus-prefixed-formatter).
    * [`zalgo`](https://github.com/aybabtme/logzalgo).
    
    
* 其他支持工具

    * [Logrus Mate](https://github.com/gogap/logrus_mate)这是一个logrus的管理工具，你可以通过配置文件来初始化logger的层级,Hook和格式。而且在不同的环境可以自动调整。
    * [Logrus Viper Helper](https://github.com/heirko/go-contrib/tree/master/logrusHelper)这是logrus的简化版 [sample](https://github.com/heirko/iris-contrib/blob/master/middleware/logrus-logger/example)
    * [Go-colorable](https://github.com/mattn/go-colorable)这将使Windows下的日志显示多色彩化。
* 致命错误处理

你可以注册自己的错误处理函数，对某类型异常进行自定义处理
```go
...
handler := func() {
  // your code...
}
logrus.RegisterExitHandler(handler)
...
```

* 线程安全

默认Logrus是线程安全的，但是当你确定自己的Log没有线程问题时，可以调用  `logger.SetNoLock()` 来解除锁定。

**基本流程**

```go
package main

import (
  log "github.com/Sirupsen/logrus"
)

func main() {
  log.WithFields(log.Fields{
    "animal": "walrus",
  }).Info("A walrus appears")
}
```

**推荐的日志HOOK库**

| Hook  | Description |
| ----- | ----------- |
| [Airbrake "legacy"](https://github.com/gemnasium/logrus-airbrake-legacy-hook) | Send errors to an exception tracking service compatible with the Airbrake API V2. Uses [`airbrake-go`](https://github.com/tobi/airbrake-go) behind the scenes. |
| [Airbrake](https://github.com/gemnasium/logrus-airbrake-hook) | Send errors to the Airbrake API V3. Uses the official [`gobrake`](https://github.com/airbrake/gobrake) behind the scenes. |
| [Amazon Kinesis](https://github.com/evalphobia/logrus_kinesis) | Hook for logging to [Amazon Kinesis](https://aws.amazon.com/kinesis/) |
| [Amqp-Hook](https://github.com/vladoatanasov/logrus_amqp) | Hook for logging to Amqp broker (Like RabbitMQ) |
| [Bugsnag](https://github.com/Shopify/logrus-bugsnag/blob/master/bugsnag.go) | Send errors to the Bugsnag exception tracking service. |
| [DeferPanic](https://github.com/deferpanic/dp-logrus) | Hook for logging to DeferPanic |
| [ElasticSearch](https://github.com/sohlich/elogrus) | Hook for logging to ElasticSearch|
| [Fluentd](https://github.com/evalphobia/logrus_fluent) | Hook for logging to fluentd |
| [Go-Slack](https://github.com/multiplay/go-slack) | Hook for logging to [Slack](https://slack.com) |
| [Graylog](https://github.com/gemnasium/logrus-graylog-hook) | Hook for logging to [Graylog](http://graylog2.org/) |
| [Hiprus](https://github.com/nubo/hiprus) | Send errors to a channel in hipchat. |
| [Honeybadger](https://github.com/agonzalezro/logrus_honeybadger) | Hook for sending exceptions to Honeybadger |
| [InfluxDB](https://github.com/Abramovic/logrus_influxdb) | Hook for logging to influxdb |
| [Influxus](http://github.com/vlad-doru/influxus) | Hook for concurrently logging to [InfluxDB](http://influxdata.com/) |
| [Journalhook](https://github.com/wercker/journalhook) | Hook for logging to `systemd-journald` |
| [KafkaLogrus](https://github.com/goibibo/KafkaLogrus) | Hook for logging to kafka |
| [LFShook](https://github.com/rifflock/lfshook) | Hook for logging to the local filesystem |
| [Logentries](https://github.com/jcftang/logentriesrus) | Hook for logging to [Logentries](https://logentries.com/) |
| [Logentrus](https://github.com/puddingfactory/logentrus) | Hook for logging to [Logentries](https://logentries.com/) |
| [Logmatic.io](https://github.com/logmatic/logmatic-go) | Hook for logging to [Logmatic.io](http://logmatic.io/) |
| [Logrusly](https://github.com/sebest/logrusly) | Send logs to [Loggly](https://www.loggly.com/) |
| [Logstash](https://github.com/bshuster-repo/logrus-logstash-hook) | Hook for logging to [Logstash](https://www.elastic.co/products/logstash) |
| [Mail](https://github.com/zbindenren/logrus_mail) | Hook for sending exceptions via mail |
| [Mongodb](https://github.com/weekface/mgorus) | Hook for logging to mongodb |
| [NATS-Hook](https://github.com/rybit/nats_logrus_hook) | Hook for logging to [NATS](https://nats.io) |
| [Octokit](https://github.com/dorajistyle/logrus-octokit-hook) | Hook for logging to github via octokit |
| [Papertrail](https://github.com/polds/logrus-papertrail-hook) | Send errors to the [Papertrail](https://papertrailapp.com) hosted logging service via UDP. |
| [PostgreSQL](https://github.com/gemnasium/logrus-postgresql-hook) | Send logs to [PostgreSQL](http://postgresql.org) |
| [Pushover](https://github.com/toorop/logrus_pushover) | Send error via [Pushover](https://pushover.net) |
| [Raygun](https://github.com/squirkle/logrus-raygun-hook) | Hook for logging to [Raygun.io](http://raygun.io/) |
| [Redis-Hook](https://github.com/rogierlommers/logrus-redis-hook) | Hook for logging to a ELK stack (through Redis) |
| [Rollrus](https://github.com/heroku/rollrus) | Hook for sending errors to rollbar |
| [Scribe](https://github.com/sagar8192/logrus-scribe-hook) | Hook for logging to [Scribe](https://github.com/facebookarchive/scribe)|
| [Sentry](https://github.com/evalphobia/logrus_sentry) | Send errors to the Sentry error logging and aggregation service. |
| [Slackrus](https://github.com/johntdyer/slackrus) | Hook for Slack chat. |
| [Stackdriver](https://github.com/knq/sdhook) | Hook for logging to [Google Stackdriver](https://cloud.google.com/logging/) |
| [Sumorus](https://github.com/doublefree/sumorus) | Hook for logging to [SumoLogic](https://www.sumologic.com/)|
| [Syslog](https://github.com/Sirupsen/logrus/blob/master/hooks/syslog/syslog.go) | Send errors to remote syslog server. Uses standard library `log/syslog` behind the scenes. |
| [TraceView](https://github.com/evalphobia/logrus_appneta) | Hook for logging to [AppNeta TraceView](https://www.appneta.com/products/traceview/) |
| [Typetalk](https://github.com/dragon3/logrus-typetalk-hook) | Hook for logging to [Typetalk](https://www.typetalk.in/) |
| [logz.io](https://github.com/ripcurld00d/logrus-logzio-hook) | Hook for logging to [logz.io](https://logz.io), a Log as a Service using Logstash |


**推荐资料**

* [Go Doc](https://godoc.org/github.com/Sirupsen/logrus#TextFormatter)

### github.com/adamzy/cedar-go

**用途**

这是一个DAT（双数组字典树）的实现。当前版本它是非线程安全的，如果你多协程进行删插将会出现问题。

**基本流程**

使用代码如下
```go
package main

import (
    "fmt"

    "github.com/adamzy/cedar-go"
)

func main() {
    // create a new cedar trie.
    trie := cedar.New()

    // a helper function to print the id-key-value triple given trie node id
    printIdKeyValue := func(id int) {
        // the key of node `id`.
        key, _ := trie.Key(id)
        // the value of node `id`.
        value, _ := trie.Value(id)
        fmt.Printf("%d\t%s:%v\n", id, key, value)
    }

    // Insert key-value pairs.
    // The order of insertion is not important.
    trie.Insert([]byte("How many"), 0)
    trie.Insert([]byte("How many loved"), 1)
    trie.Insert([]byte("How many loved your moments"), 2)
    trie.Insert([]byte("How many loved your moments of glad grace"), 3)
    trie.Insert([]byte("姑苏"), 4)
    trie.Insert([]byte("姑苏城外"), 5)
    trie.Insert([]byte("姑苏城外寒山寺"), 6)

    // Get the associated value of a key directly.
    value, _ := trie.Get([]byte("How many loved your moments of glad grace"))
    fmt.Println(value)

    // Or, jump to the node first,
    id, _ := trie.Jump([]byte("How many loved your moments"), 0)
    // then get the key and the value
    printIdKeyValue(id)

    fmt.Println("\nPrefixMatch\nid\tkey:value")
    for _, id := range trie.PrefixMatch([]byte("How many loved your moments of glad grace"), 0) {
        printIdKeyValue(id)
    }

    fmt.Println("\nPrefixPredict\nid\tkey:value")
    for _, id := range trie.PrefixPredict([]byte("姑苏"), 0) {
        printIdKeyValue(id)
    }
}
```

将会输出

    3
    281 How many loved your moments:2
    
    PrefixMatch
    id  key:value
    262 How many:0
    268 How many loved:1
    281 How many loved your moments:2
    296 How many loved your moments of glad grace:3
    
    PrefixPredict
    id  key:value
    303 姑苏:4
    309 姑苏城外:5
    318 姑苏城外寒山寺:6

### github.com/boltdb/bolt

**用途**

Bolt是一个Go语言编写的，小型的便于嵌入的，可序列化的，事务性的，Key/Value存储数据库。
* 该项目的目标是为不需要完整的数据库（例如MySQL）的项目，提供一个简单快速可靠的数据库。
* 因为Bolt做的是低级功能，所以它的目的是简单。它的API非常少，专注于获取值和设置值这种基本功能而已。
* Bolt当前是稳定的，API已固定，文件格式也已固定，经过全单元覆盖测试盒随机黑盒测试，可以保证数据库的一致性和线程安全性。

> 注意
> * Bolt对数据库独占写锁定，所以不可多进程共享。
> * Bolt支持事务，回滚。
> * Bolt支持**IOS/Android**上使用。
> * Bolt文件记录是和CPU大小端有关的。

**核心函数简述**

* Open() 打开/创建 数据库。
* DB.Begin() 根据可写参数的值，开始读或读写事务。
* Bucket.Put() 将 Key/Value对 写入存储桶。
* Bucket.Get() 从存储桶中检索 Key/Value对。
* Tx.Commit() 将内存中的脏节点和空闲页写入磁盘保存

**基本流程**

* [官网](github.com/boltdb/bolt)
    
**推荐资料**

* [Intro to BoltDB: Painless Performant Persistence](http://npf.io/2014/07/intro-to-boltdb-painless-performant-persistence/)
* [Bolt -- an embedded key/value database for Go](https://www.progville.com/go/bolt-embedded-db-golang/)

**使用样例**
* [BoltDbWeb](https://github.com/evnix/boltdbweb) - A web based GUI for BoltDB files.
* [stow](https://github.com/djherbis/stow) -  a persistence manager for objects backed by boltdb.
* [buckets](https://github.com/joyrexus/buckets) - a bolt wrapper streamlining simple tx and key scans.
* [mbuckets](https://github.com/abhigupta912/mbuckets) - A Bolt wrapper that allows easy operations on multi level (nested) buckets.
* [Boltdb Boilerplate](https://github.com/bobintornado/boltdb-boilerplate) - Boilerplate wrapper around bolt aiming to make simple calls one-liners.
* [Storm](https://github.com/asdine/storm) - Simple and powerful ORM for BoltDB.
* [GoWebApp](https://github.com/josephspurrier/gowebapp) - A basic MVC web application in Go using BoltDB.
* [SimpleBolt](https://github.com/xyproto/simplebolt) - A simple way to use BoltDB. Deals mainly with strings.
* [Algernon](https://github.com/xyproto/algernon) - A HTTP/2 web server with built-in support for Lua. Uses BoltDB as the default database backend.
* [GoShort](https://github.com/pankajkhairnar/goShort) - GoShort is a URL shortener written in Golang and BoltDB for persistent key/value storage and for routing it's using high performent HTTPRouter.
* [torrent](https://github.com/anacrolix/torrent) - Full-featured BitTorrent client package and utilities in Go. BoltDB is a storage backend in development.
* [bolter](https://github.com/hasit/bolter) - Command-line app for viewing BoltDB file in your terminal.
* [BoltHold](https://github.com/timshannon/bolthold) - An embeddable NoSQL store for Go types built on BoltDB

### github.com/coreos/etcd

**用途**

ETCD是一个Go编写的，分布式的Key/Value存储软件，用于配置共享和服务发现。
它的特点包括：
* 简单：面向用户，定义明确的API（gRPC）
* 安全：自动化TLS客户端证书身份验证
* 快速：基本测试10000次写入/秒
* 可靠：基于Raft一致性算法来管理高可用性记录

**基本流程**

1. 进入 `./bin/etcd` 执行它，此时默认会监听2379端口等待客户端请求，监听2380端口等待其他节点通信。
1. 然后我们来测试加入Key/Value值，然后访问它

        ETCDCTL_API=3 etcdctl put MyKey "helloworld"
        ETCDCTL_API=3 etcdctl get MyKey
    
1. 这就完成了

**推荐资料**
* [官网](https://github.com/coreos/etcd)
* [ETCD控制台](https://github.com/coreos/etcd/tree/master/etcdctl)
* [ETCDDemo](https://github.com/coreos/etcd/blob/master/Documentation/demo.md)
* [安装和部署ETCD](http://play.etcd.io/install)
* [下载ETCD](https://github.com/coreos/etcd/releases/)

### github.com/davecgh/go-spew/spew

**用途**

Go-Spew是一个对Go数据进行格式化输出的库，它通常在调试时候使用。

**简要说明**

如果打印完整信息，可以使用如下代码
```go
spew.Dump(myVar1, myVar2, ...)
spew.Fdump(someWriter, myVar1, myVar2, ...)
str := spew.Sdump(myVar1, myVar2, ...)
```
如果想自己执行格式化方式，可以使用如下代码
```go
spew.Printf("myVar1: %v -- myVar2: %+v", myVar1, myVar2)
spew.Printf("myVar3: %#v -- myVar4: %#+v", myVar3, myVar4)
spew.Fprintf(someWriter, "myVar1: %v -- myVar2: %+v", myVar1, myVar2)
spew.Fprintf(someWriter, "myVar3: %#v -- myVar4: %#+v", myVar3, myVar4)
```
**使用范例**
```go
    package main
    import (
        "fmt"
        "html"
        "net/http"
        "github.com/davecgh/go-spew/spew"
    )
    func handler(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        fmt.Fprintf(w, "Hi there, %s!", r.URL.Path[1:])
        fmt.Fprintf(w, "<!--\n" + html.EscapeString(spew.Sdump(w)) + "\n-->")
    }
    func main() {
        http.HandleFunc("/", handler)
        http.ListenAndServe(":8080", nil)
    }
```
那么将会输出Dump信息如下：

    (main.Foo) {
     unexportedField: (*main.Bar)(0xf84002e210)({
      flag: (main.Flag) flagTwo,
      data: (uintptr) <nil>
     }),
     ExportedField: (map[interface {}]interface {}) {
      (string) "one": (bool) true
     }
    }
    ([]uint8) {
     00000000  11 12 13 14 15 16 17 18  19 1a 1b 1c 1d 1e 1f 20  |............... |
     00000010  21 22 23 24 25 26 27 28  29 2a 2b 2c 2d 2e 2f 30  |!"#$%&'()*+,-./0|
     00000020  31 32                                             |12|
    }
    
    ```
    
**推荐资料**
* [博客](https://web.archive.org/web/20160304013555/https://blog.cyphertite.com/go-spew-a-journey-into-dumping-go-data-structures/)
* [Go doc](http://godoc.org/github.com/davecgh/go-spew/spew)

### github.com/eapache/go-resiliency/breaker

**用途**

这是一个简单的熔断器模式。

**基本流程**

创建一个熔断器需要三个参数：
* 错误阀值（用来开启熔断）
* 成功阀值（用来关闭熔断）
* 熔断时间（熔断保持多久）

```go
b := breaker.New(3, 1, 5*time.Second)

for {
    result := b.Run(func() error {
        // 尝试和外部服务通信，如果通信失败则返回错误
        return nil
    })

    switch result {
    case nil:
        // 通信成功
    case breaker.ErrBreakerOpen:
        // 当前被熔断状态
    default:
        // 其他一些错误
    }
}
```
**推荐资料**

* [熔断器模式](http://blog.csdn.net/joeyangy/article/details/43112687)

### github.com/eapache/queue

**用途**

一个线程不安全的，但内存占用和时间效率很高的环形缓冲区队列。

### github.com/golang/protobuf/proto

**用途**

它是Google Protobuf的Go语言版本。

**使用说明**
```go
    package main

    import (
        "log"

        "github.com/golang/protobuf/proto"
        "path/to/example"
    )

    func main() {
        test := &example.Test {
            Label: proto.String("hello"),
            Type:  proto.Int32(17),
            Reps:  []int64{1, 2, 3},
            Optionalgroup: &example.Test_OptionalGroup {
                RequiredField: proto.String("good bye"),
            },
        }
        data, err := proto.Marshal(test)
        if err != nil {
            log.Fatal("marshaling error: ", err)
        }
        newTest := &example.Test{}
        err = proto.Unmarshal(data, newTest)
        if err != nil {
            log.Fatal("unmarshaling error: ", err)
        }
        // Now test and newTest contain the same data.
        if test.GetLabel() != newTest.GetLabel() {
            log.Fatalf("data mismatch %q != %q", test.GetLabel(), newTest.GetLabel())
        }
        // etc.
    }
```

**推荐资料**
* [Protobuf说明](https://developers.google.com/protocol-buffers/)

### github.com/golang/snappy

**用途**

Google Snappy压缩算法的Go语言版本。该算法的压缩速率极快，但压缩结果会比zip大20%-80%左右。

### github.com/huichen/sego

**用途**

Go中文分词库

**基本说明**

* <a href="https://github.com/huichen/sego/blob/master/dictionary.go">词典</a>用双数组trie（Double-Array Trie）实现，
* <a href="https://github.com/huichen/sego/blob/master/segmenter.go">分词器</a>算法为基于词频的最短路径加动态规划。
* 支持普通和搜索引擎两种分词模式，支持用户词典、词性标注，可运行<a href="https://github.com/huichen/sego/blob/master/server/server.go">JSON RPC服务</a>。
* 分词速度<a href="https://github.com/huichen/sego/blob/master/tools/benchmark.go">单线程</a>9MB/s，<a href="https://github.com/huichen/sego/blob/master/tools/goroutines.go">goroutines并发</a>42MB/s（8核Macbook Pro）。

**使用说明**
```go
package main

import (
	"fmt"
	"github.com/huichen/sego"
)

func main() {
	// 载入词典
	var segmenter sego.Segmenter
	segmenter.LoadDictionary("github.com/huichen/sego/data/dictionary.txt")

	// 分词
	text := []byte("中华人民共和国中央人民政府")
	segments := segmenter.Segment(text)
  
	// 处理分词结果
	// 支持普通模式和搜索模式两种分词，见代码中SegmentsToString函数的注释。
	fmt.Println(sego.SegmentsToString(segments, false)) 
}
```
### github.com/jroimartin/gocui

**用途**

创建控制台用户界面的小型Go库

**特性说明**

* 非常简单的API
* 直接在窗口中实现接口 io.ReadWriter
* 支持重叠视图
* 支持热键绑定
* 支持鼠标操作
* 彩色文本
* 可定制样式
* 提供易于构建可复用的小部件

**使用说明**
```go
package main

import (
    "fmt"
    "log"

    "github.com/jroimartin/gocui"
)

func main() {
    g, err := gocui.NewGui(gocui.OutputNormal)
    if err != nil {
        log.Panicln(err)
    }
    defer g.Close()

    g.SetManagerFunc(layout)

    if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
        log.Panicln(err)
    }

    if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
        log.Panicln(err)
    }
}

func layout(g *gocui.Gui) error {
    maxX, maxY := g.Size()
    if v, err := g.SetView("hello", maxX/2-7, maxY/2, maxX/2+7, maxY/2+2); err != nil {
        if err != gocui.ErrUnknownView {
            return err
        }
        fmt.Fprintln(v, "Hello world!")
    }
    return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
    return gocui.ErrQuit
}
```
**推荐资料**

* [Go doc](https://godoc.org/github.com/jroimartin/gocui)
    
**使用参考** 

* [Komanda CLI](https://github.com/mephux/komanda-cli): IRC Client For Developers.
* [Vuls](https://github.com/future-architect/vuls): Agentless vulnerability scanner for Linux/FreeBSD.
* [wuzz](https://github.com/asciimoo/wuzz): Interactive cli tool for HTTP inspection.
* [httplab](https://github.com/gchaincl/httplab): Interactive web server.
* [domainr](https://github.com/MichaelThessel/domainr): Tool that checks the availability of domains based on keywords.

### github.com/peterbourgon/g2s

**用途**

Statsd的封装库。
* 可以将简单的统计信息转发给StatsD服务器
* 提供了一个StastD的对象，并为每个支持的StatsD统计类型进行一些便利的封装

**使用说明**

如果你需要为自己的统计指标增加前缀，可以在实例化StatsD结构时使用 `DialWithPrefix` :
```go
s, err := g2s.Dial("udp", "statsd-server:8125")
if err != nil {
    // do something
}

s.Counter(1.0, "my.silly.counter", 1)
s.Timing(1.0, "my.silly.slow-process", time.Since(somethingBegan))
s.Timing(0.2, "my.silly.fast-process", 7*time.Millisecond)
s.Gauge(1.0, "my.silly.status", "green")
```
如果你使用标准UDP连接StatsD服务器，所有的 'update' 类函数都是协程安全的。所以，你在一个单独线程调用。
```go
s, err := g2s.DialWithPrefix("udp", "statsd-server:8125", "my-cool-prefix")
if err != nil {
    // do something
}

s.Counter(1.0, "this.gets.prefixed", 1)
```

### github.com/oschwald/maxminddb-golang

**用途**

这是MaxMind DB格式的读取器库。
* 它用来读取[GeoLite2](http://dev.maxmind.com/geoip/geoip2/geolite2/) 和
       [GeoIP2](https://www.maxmind.com/en/geoip2-databases)的数据库文件

**推荐资料**

* [Go doc](http://godoc.org/github.com/oschwald/maxminddb-golang)
    
### github.com/tealeg/xlsx

**用途**

XLSX是一个读取和写入Micrsoft Excel生成的xlsx格式的库。

**使用说明**
* 读取数据
```go
package main

import (
    "fmt"
    "github.com/tealeg/xlsx"
)

func main() {
    excelFileName := "/home/tealeg/foo.xlsx"
    xlFile, err := xlsx.OpenFile(excelFileName)
    if err != nil {
        ...
    }
    for _, sheet := range xlFile.Sheets {
        for _, row := range sheet.Rows {
            for _, cell := range row.Cells {
                text, _ := cell.String()
                fmt.Printf("%s\n", text)
            }
        }
    }
}
```
* 写入数据
```go
package main

import (
    "fmt"
    "github.com/tealeg/xlsx"
)

func main() {
    var file *xlsx.File
    var sheet *xlsx.Sheet
    var row *xlsx.Row
    var cell *xlsx.Cell
    var err error

    file = xlsx.NewFile()
    sheet, err = file.AddSheet("Sheet1")
    if err != nil {
        fmt.Printf(err.Error())
    }
    row = sheet.AddRow()
    cell = row.AddCell()
    cell.Value = "I am a cell!"
    err = file.Save("MyXLSXFile.xlsx")
    if err != nil {
        fmt.Printf(err.Error())
    }
}
```

**推荐资料**

* [Go doc](https://godoc.org/github.com/tealeg/xlsx)
* [xlsx2csv](https://github.com/tealeg/xlsx2csv)
    
### github.com/klauspost/cpuid

**用途**

用来获取当前程序所在CPU信息的库。
* 它没有任何外部C代码依赖

**使用范例**：
```go
package main

import (
    "fmt"
    "github.com/klauspost/cpuid"
)

func main() {
    // Print basic CPU information:
    fmt.Println("Name:", cpuid.CPU.BrandName)
    fmt.Println("PhysicalCores:", cpuid.CPU.PhysicalCores)
    fmt.Println("ThreadsPerCore:", cpuid.CPU.ThreadsPerCore)
    fmt.Println("LogicalCores:", cpuid.CPU.LogicalCores)
    fmt.Println("Family", cpuid.CPU.Family, "Model:", cpuid.CPU.Model)
    fmt.Println("Features:", cpuid.CPU.Features)
    fmt.Println("Cacheline bytes:", cpuid.CPU.CacheLine)
    fmt.Println("L1 Data Cache:", cpuid.CPU.Cache.L1D, "bytes")
    fmt.Println("L1 Instruction Cache:", cpuid.CPU.Cache.L1D, "bytes")
    fmt.Println("L2 Cache:", cpuid.CPU.Cache.L2, "bytes")
    fmt.Println("L3 Cache:", cpuid.CPU.Cache.L3, "bytes")

    // Test if we have a specific feature:
    if cpuid.CPU.SSE() {
        fmt.Println("We have Streaming SIMD Extensions")
    }
}
```
这将输出如下内容：

    Name: Intel(R) Core(TM) i5-2540M CPU @ 2.60GHz
    PhysicalCores: 2
    ThreadsPerCore: 2
    LogicalCores: 4
    Family 6 Model: 42
    Features: CMOV,MMX,MMXEXT,SSE,SSE2,SSE3,SSSE3,SSE4.1,SSE4.2,AVX,AESNI,CLMUL
    Cacheline bytes: 64
    We have Streaming SIMD Extensions

### github.com/klauspost/crc32

**用途**

用来替换标准库中的 `hash/crc32` .
* 它在 x64 平台上具有SSE 4.2优化功能，可以性能提高10倍。

### github.com/klauspost/reedsolomon

**用途**

RS（里德-所罗门）纠错算法。

**基本描述**

在数据传输过程中，可能会有部分数据丢失或收到干扰出错，此时RS算法就是一种常见的算法用来恢复错误的数据。
* 它会生成一些额外的纠错信息，所以会使整个数据量增大。
* 它不仅仅是签名，因为它不仅仅可以校验数据完整性，侦查错误，还可以纠正错误。

**推荐资料**

* [开发者博客](https://www.backblaze.com/blog/reed-solomon/)
* [Backblaze Open Sources Reed-Solomon Erasure Coding Source Code](https://www.backblaze.com/blog/reed-solomon/).
* [JavaReedSolomon](https://github.com/Backblaze/JavaReedSolomon). Compatible java library by Backblaze.
* [reedsolomon-c](https://github.com/jannson/reedsolomon-c). C version, compatible with output from this package.
* [Reed-Solomon Erasure Coding in Haskell](https://github.com/NicolasT/reedsolomon). Haskell port of the package with similar performance.
* [go-erasure](https://github.com/somethingnew2-0/go-erasure). A similar library using cgo, slower in my tests.
* [rsraid](https://github.com/goayame/rsraid). A similar library written in Go. Slower, but supports more shards.
* [Screaming Fast Galois Field Arithmetic](http://www.snia.org/sites/default/files2/SDC2013/presentations/NewThinking/EthanMiller_Screaming_Fast_Galois_Field%20Arithmetic_SIMD%20Instructions.pdf). Basis for SSE3 optimizations.

### github.com/mattn/go-runewidth

**用途**

获取字符或字符串宽度的函数库，支持多语言。

**使用范例**：
```go
runewidth.StringWidth("つのだ☆HIRO") == 12
```
### github.com/nsf/termbox-go

**用途**

是一套简易的，跨平台的，基于文本操作的用户界面。

**推荐资料**
 - [Go doc](http://godoc.org/github.com/nsf/termbox-go)
 - [godit](https://github.com/nsf/godit) is an emacsish lightweight text editor written using termbox.
 - [gomatrix](https://github.com/GeertJohan/gomatrix) connects to The Matrix and displays its data streams in your terminal.
 - [gotetris](https://github.com/jjinux/gotetris) is an implementation of Tetris.
 - [sokoban-go](https://github.com/rn2dy/sokoban-go) is an implementation of sokoban game.
 - [hecate](https://github.com/evanmiller/hecate) is a hex editor designed by Satan.
 - [httopd](https://github.com/verdverm/httopd) is top for httpd logs.
 - [mop](https://github.com/michaeldv/mop) is stock market tracker for hackers.
 - [termui](https://github.com/gizak/termui) is a terminal dashboard.
 - [termloop](https://github.com/JoelOtter/termloop) is a terminal game engine.
 - [xterm-color-chart](https://github.com/kutuluk/xterm-color-chart) is a XTerm 256 color chart.
 - [gocui](https://github.com/jroimartin/gocui) is a minimalist Go library aimed at creating console user interfaces.
 - [dry](https://github.com/moncho/dry) is an interactive cli to manage Docker containers.
 - [pxl](https://github.com/ichinaski/pxl) displays images in the terminal.
 - [snake-game](https://github.com/DyegoCosta/snake-game) is an implementation of the Snake game.
 - [gone](https://github.com/guillaumebreton/gone) is a CLI pomodoro? timer.
 - [Spoof.go](https://github.com/sabey/spoofgo) controllable movement spoofing from the cli
 - [lf](https://github.com/gokcehan/lf) is a terminal file manager
 - [rat](https://github.com/ericfreese/rat) lets you compose shell commands to build terminal applications.
 - [httplab](https://github.com/gchaincl/httplab) An interactive web server.

### github.com/pkg/errors

**用途**

是一个错误处理的加强包

**使用范例**：
- 一般的错误处理如下
```
if err != nil {
        return err
}
```
本库允许你对错误添加上下文
```go
_, err := ioutil.ReadAll(r)
if err != nil {
        return errors.Wrap(err, "read failed")
}
```
因为错误可能生成一个错误栈，可以使用Cause来检索最原始的错误
```go
switch err := errors.Cause(err).(type) {
case *MyError:
        // handle specifically
default:
        // unknown error
}
```
指定错误输出格式
```
%s    输出错误。如果错误有Cause处理，将进行递归输出
%v    参见 %s
%+v   加强信息，将输出错误的调用堆栈
```
输出错误时的堆栈
在调用Wrap,New,Errorf,Wrapf时，会记录一个错误调用堆栈，可以将其输出出来
```go
if err, ok := err.(stackTracer); ok {
        for _, f := range err.StackTrace() {
                fmt.Printf("%+s:%d", f)
        }
}
```

**推荐资料**
- [Go doc](https://godoc.org/github.com/pkg/errors)

### github.com/pierrec/lz4

**用途**

LZ4压缩和解压缩库。
LZ4是速度最快的压缩和解压缩库。

### github.com/pierrec/xxHash/xxHash32

**用途**

是一个非常快速的散列算法库。

**使用范例**：
```go
import (
		"fmt"

		"github.com/pierrec/xxHash/xxHash32"
	)

 	x := xxHash32.New(0xCAFE) // hash.Hash32
	x.Write([]byte("abc"))
	x.Write([]byte("def"))
	fmt.Printf("%x\n", x.Sum32())

	x.Reset()
	x.Write([]byte("abc"))
	fmt.Printf("%x\n", x.Sum32())
```
**推荐资料**
* [散列算法速度比较](https://github.com/Cyan4973/xxHash/)

### github.com/rcrowley/go-metrics

**用途**

一个类似Java Metrics的便捷的程序性能监测库。

**使用范例**：
- 创建并更新Metrics
```go
c := metrics.NewCounter()
metrics.Register("foo", c)
c.Inc(47)

g := metrics.NewGauge()
metrics.Register("bar", g)
g.Update(47)

r := NewRegistry()
g := metrics.NewRegisteredFunctionalGauge("cache-evictions", r, func() int64 { return cache.getEvictionsCount() })

s := metrics.NewExpDecaySample(1028, 0.015) // or metrics.NewUniformSample(1028)
h := metrics.NewHistogram(s)
metrics.Register("baz", h)
h.Update(47)

m := metrics.NewMeter()
metrics.Register("quux", m)
m.Mark(47)

t := metrics.NewTimer()
metrics.Register("bang", t)
t.Time(func() {})
t.Update(47)
```
- 线程安全的更新方式：将上文的 `Register` 替换成 `GetOrRegisterTimer` 即可。
```go
t := metrics.GetOrRegisterTimer("account.create.latency", nil)
t.Time(func() {})
t.Update(47)
```
- 以可读的Log方式定期记录指标误差
```go
go metrics.Log(metrics.DefaultRegistry, 5 * time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
```
- 以便于分析的方式将指标定期记录到syslog
```go
w, _ := syslog.Dial("unixgram", "/dev/log", syslog.LOG_INFO, "metrics")
go metrics.Syslog(metrics.DefaultRegistry, 60e9, w)
```
- 使用Graphite客户端定期更新指标
```go
import "github.com/cyberdelia/go-metrics-graphite"

addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:2003")
go graphite.Graphite(metrics.DefaultRegistry, 10e9, "metrics", addr)
```
- 定期向InfluxDB更新指标
```go
import "github.com/vrischmann/go-metrics-influxdb"

go influxdb.Influxdb(metrics.DefaultRegistry, 10e9, &influxdb.Config{
    Host:     "127.0.0.1:8086",
    Database: "metrics",
    Username: "test",
    Password: "test",
})
```
- 定期向[Librato](https://github.com/mihasya/go-metrics-librato)客户端更新指标
```go
import "github.com/mihasya/go-metrics-librato"

go librato.Librato(metrics.DefaultRegistry,
    10e9,                  // interval
    "example@example.com", // account owner email address
    "token",               // Librato API token
    "hostname",            // source
    []float64{0.95},       // percentiles to send
    time.Millisecond,      // time unit
)
```
- 定期向StatHat更新指标
```go
import "github.com/rcrowley/go-metrics/stathat"

go stathat.Stathat(metrics.DefaultRegistry, 10e9, "example@example.com")
```

**推荐资料**
* [Go doc](https://godoc.org/github.com/rcrowley/go-metrics)
* Librato - [https://github.com/mihasya/go-metrics-librato](https://github.com/mihasya/go-metrics-librato)
* Graphite - [https://github.com/cyberdelia/go-metrics-graphite](https://github.com/cyberdelia/go-metrics-graphite)
* InfluxDB - [https://github.com/vrischmann/go-metrics-influxdb](https://github.com/vrischmann/go-metrics-influxdb)
* Ganglia - [https://github.com/appscode/metlia](https://github.com/appscode/metlia)
* Prometheus - [https://github.com/deathowl/go-metrics-prometheus](https://github.com/deathowl/go-metrics-prometheus)

### github.com/ugorji/go/codec

**用途**

**核心组成**
 - **MessagePack** 是一种高效的二进制序列化格式，它类似JSON，但是更加快速更加小。
    * [MsgPack官网](http://msgpack.org/)
    * [MsgPack GitHub页面](https://github.com/msgpack/website)
    * [MsgPack格式](https://github.com/msgpack/msgpack/blob/master/spec.md)
 - **Binc** 是一种轻量级的，紧凑的，无限制的，精确地，二进制的，高性能的，功能丰富的，语言无关的，可扩展的数据交换格式。
    * 它遵循[语义化版本控制规范2.0.0](http://semver.org/lang/zh-CN/)
    * [格式说明](https://github.com/ugorji/binc/blob/master/SPEC.md)
    * [推荐文章1](http://www.ugorji.net/blog/announcing-binc-data-interchange-format)
    * [推荐文章2](http://www.ugorji.net/blog/update-on-binc-data-interchange-format)
 - **CBOR** 是一个对象编码格式，类似于一个二进制版的JSON，但支持更多特性。
    * [格式标准](http://cbor.io/)
    
**使用范例**：

RPC客户端和服务器代码都已经实现，因此这个包可以直接和标准 net/rpc 包一起用。
```go
// create and configure Handle
var (
  bh codec.BincHandle
  mh codec.MsgpackHandle
  ch codec.CborHandle
)

mh.MapType = reflect.TypeOf(map[string]interface{}(nil))

// configure extensions
// e.g. for msgpack, define functions and enable Time support for tag 1
// mh.SetExt(reflect.TypeOf(time.Time{}), 1, myExt)

// create and use decoder/encoder
var (
  r io.Reader
  w io.Writer
  b []byte
  h = &bh // or mh to use msgpack
)

dec = codec.NewDecoder(r, h)
dec = codec.NewDecoderBytes(b, h)
err = dec.Decode(&v)

enc = codec.NewEncoder(w, h)
enc = codec.NewEncoderBytes(&b, h)
err = enc.Encode(v)

//RPC Server
go func() {
    for {
        conn, err := listener.Accept()
        rpcCodec := codec.GoRpc.ServerCodec(conn, h)
        //OR rpcCodec := codec.MsgpackSpecRpc.ServerCodec(conn, h)
        rpc.ServeCodec(rpcCodec)
    }
}()

//RPC Communication (client side)
conn, err = net.Dial("tcp", "localhost:5555")
rpcCodec := codec.GoRpc.ClientCodec(conn, h)
//OR rpcCodec := codec.MsgpackSpecRpc.ClientCodec(conn, h)
client := rpc.NewClientWithCodec(rpcCodec)
```

**推荐资料**
* [Go doc](http://godoc.org/github.com/ugorji/go/codec)
* [使用向导](http://ugorji.net/blog/go-codec-primer)

### github.com/xtaci/logrushooks

**用途**

一个简单的Logrus的Hook库。

### github.com/xtaci/kcp-go

**用途**

快速的可靠的有序的UDP库

**特点**
* 对多人游戏，在线视频流做了很好的优化
* 比TCP浪费10%-20%的带宽，但是延迟降低了30%-40%
* TCP丢包后，是从丢失的包开始，之后的包全部重传；KCP则是真正传输丢失的包
* 兼容net.Conn和net.Listener，便于使用
* 支持前向纠错，支持Reed-Solomon编码
* 包支持AES,TEA,3DES,Blowfish,Cast5,Salsa20等加密

**其他说明**

    +--------------+
    | SESSION      |
    +--------------+
    | KCP(ARQ)     |
    +--------------+
    | FEC          |
    +--------------+
    | CRYPTO       |
    +--------------+
    | UDP(PACKET)  |
    +--------------+
    | IP           |
    +--------------+
    | LINK         |
    +--------------+
    | PHY          |
    +--------------+
    (KCP-GO七层网络模型)
    
     +---------------------------------------------------------------------------------------------------------+
     | Nonce(16-bits) | Crc32(4-bits) | FEC SeqID(4-bits) | FEC Type(2-bits) |||  SIZE(2-bits) | KCP(SIZE - 2) |
     +---------------------------------------------------------------------------------------------------------+

**使用范例**
```go
    // Client:
        kcpConn, err := kcp.DialWithOptions("192.168.1.1:8888", nil, 10, 3)
    // Server:
        lis, err := kcp.ListenWithOptions(":8888", nil, 10, 3)
```

**相关资料**
* [下载版本](https://github.com/xtaci/kcp-go/releases)
* [性能测评](https://github.com/libinzhangyuan/reliable_udp_bench_mark/blob/master/bench_mark.md)
* [Go Doc](https://godoc.org/github.com/xtaci/kcp-go)
* [客户端完整代码样例](https://github.com/xtaci/kcptun/blob/master/client/main.go)
* [服务器完整代码样例](https://github.com/xtaci/kcptun/blob/master/server/main.go)
* [推荐参考 Kcptun](https://github.com/xtaci/kcptun)

### gopkg.in/mgo.v2

**用途**

MangoDB的Go语言驱动库。

**使用范例**：
```go

package main

import (
        "fmt"
	"log"
        "gopkg.in/mgo.v2"
        "gopkg.in/mgo.v2/bson"
)

type Person struct {
        Name string
        Phone string
}

func main() {
        session, err := mgo.Dial("server1.example.com,server2.example.com")
        if err != nil {
                panic(err)
        }
        defer session.Close()

        // Optional. Switch the session to a monotonic behavior.
        session.SetMode(mgo.Monotonic, true)

        c := session.DB("test").C("people")
        err = c.Insert(&Person{"Ale", "+55 53 8116 9639"},
	               &Person{"Cla", "+55 53 8402 8510"})
        if err != nil {
                log.Fatal(err)
        }

        result := Person{}
        err = c.Find(bson.M{"name": "Ale"}).One(&result)
        if err != nil {
                log.Fatal(err)
        }

        fmt.Println("Phone:", result.Phone)
}
```

**推荐资料**
* [官网](http://labix.org/mgo)
* [Mgo文档](http://gopkg.in/mgo.v2)
* [MgoBson文档](http://gopkg.in/mgo.v2/bson)
* [MgoTxn文档](http://gopkg.in/mgo.v2/txn)

### gopkg.in/vmihailenco/msgpack.v2

**用途**

msgpack是一个高效的二进制序列化格式，支持多种语言之间进行数据交换。
* 它非常类似boost的Serialization, google的protobuf等。
* 但效率比protobuf快四倍，比json快10倍

**使用范例**：
```go
func ExampleMarshal() {
	type Item struct {
		Foo string
	}

	b, err := msgpack.Marshal(&Item{Foo: "bar"})
	if err != nil {
		panic(err)
	}

	var item Item
	err = msgpack.Unmarshal(b, &item)
	if err != nil {
		panic(err)
	}
	fmt.Println(item.Foo)
	// Output: bar
}
```

**推荐资料**
- [Github](https://github.com/vmihailenco/msgpack/tree/v2.8.4)
- [GO DOC](https://godoc.org/gopkg.in/vmihailenco/msgpack.v2)
- [例子](https://godoc.org/gopkg.in/vmihailenco/msgpack.v2#pkg-examples)
- [MsgPack官网](http://msgpack.org/)

### google.golang.org/grpc

**用途**

gRPC是一个高性能，开源和通用的远程过程调用(RPC)框架，面向移动和HTTP/2设计，支持大部分语言。

**基本流程**

和大部分RPC系统类似，gRPC客户端可以像调用本地对象一样直接调用另外一台不同机器上服务端应用：
1. gRPC定义一个服务，指定其能够被远程调用的方法，包括函数和返回类型。
1. 在服务端实现该接口，并运行一个gRPC服务器来处理客户端调用。
1. 客户端拥有一个和服务端一样的方法的存根，并通过HTTP/2向服务端发出请求Request，服务端给予即使Response。
1. gRPC使用的是ProtoBuffers进行通讯，其序列化和反序列是非常高效的。

**推荐资料**

* [RPC简介](http://doc.oschina.net/grpc?t=60133)
    

### gopkg.in/urfave/cli.v2

**用途**

构建控制台命令行的库

**使用范例**：
```go
    package main
    import (
      "fmt"
      "os"
      "github.com/urfave/cli"
    )
    
    func main() {
      app := cli.NewApp()
      app.Name = "greet"
      app.Usage = "fight the loneliness!"
      app.Action = func(c *cli.Context) error {
        fmt.Println("Hello friend!")
        return nil
      }
      app.Run(os.Args)
    }
```

然后在命令行就可以获得如下支持

```go
    $ greet help
    NAME:
        greet - fight the loneliness!
    USAGE:
        greet [global options] command [command options] [arguments...]
    VERSION:
        0.0.0
    COMMANDS:
        help, h  Shows a list of commands or help for one command
    GLOBAL OPTIONS
        --version Shows version information
```

**用途**

用于共享配置和服务发现的，分布式的KeyValue存储系统

**使用范例**
```go
    package main
    import (
        "log"
        "github.com/coreos/go-etcd/etcd"
    )
    func main() {
        host := []string{"http://127.0.0.1:2379"}
        client := etcd.NewClient(host)
        // 添加KeyValue值
        if _, err := client.Set("/foo", "bar", 0); err != nil {
            log.Fatal(err)
        }
    }
```
### github.com/golang/x/net/http2

**用途**

使用**HTTP/2**协议作为服务端内部通讯协议

**基本流程**

* [参考使用样例](https://github.com/golang/net/blob/master/http2/h2demo)

**推荐资料**

* [Http2测试](https://http2.golang.org)
* [使用样例](https://github.com/golang/net/blob/master/http2/h2demo)

# 其他补充

## HTTP/2协议

### 概述

**HTTP/2** 依然是一个基于TCP的面向请求的协议。客户端向服务端发送请求，服务端生成响应并将其发回。（ServerPush是个小例外）

1. HTTP/2是HTTP的升级替代品，而不是一个协议的基础重写。在状态代码和语义上两者是相同的，HTTP/2兼容HTTP/1.x。
1. 它的设计目的是比HTTP更加高性能，低延迟，更充分的利用网络流量资源的让用户通过浏览器来访问指定网站。

### 比较
1. **HTTP/2**基于**SPDY/2**为基础开发，但和SPDY仍有不同的地方如下：

* HTTP/2 支持明文HTTP传输，而SPDY强制使用HTTPS。
* HTTP/2 消息头压缩算法采用[HPACK](http2.github.io/http2-spec/compression.html)，而SPDY消息头压缩算法采用的是[DEFLATE](https://zh.wikipedia.org/wiki/DEFLATE)。

1. **HTTP/2**和**HTTP/1.x**的比较

* HTTP/2 采用二进制格式进行数据传输； HTTP/1.x 采用文本格式进行数据传输。HTTP/2 在协议的解析和优化扩展上有更多优势和可能，但调试麻烦一些。
* HTTP/2 对消息头采用HPACK压缩，节省了消息头占用的网络流量； HTTP/1.x 则不进行压缩，会有冗余头信息的流量浪费。
* HTTP/2 支持多路复用，即将多有请求通过一个TCP连接并发完成； HTTP/1.x 则不支持。
* HTTP/2 支持消息的优先级和流量控制；Http/1.x 则不支持。
* HTTP/2 支持Server Push，例如服务器端可以主动将JS和CSS文件推送给客户端，而不需要客户端解析HTML再发送请求了；Http/1.x 不支持Server Push.

### 实现原理

**二进制Frame和特性说明**

HTTP/2 的底层数据格式是Frame，它类似于TCP里面的二进制数据包一样。其基本格式如下

        +-----------------------------------------------+
        |                   Length (24)                 |
        +---------------+---------------+---------------+
        |   Type (8)    |   Flags (8)   |
        +-+-------------+---------------+-------------------------------+
        |R|                 Stream Identifier (31)                      |
        +=+=============================================================+
        |                   Frame Payload (0...)                      ...
        +---------------------------------------------------------------+
        
        1：Frame的格式
            1.1：Length 表示Frame Payload部分的数据长度。
            1.2：其Headers部分长度 = Length + Type + Flags + R + Stream Identifier，该值是固定的 24 + 8 + 8 + 1 + 31 = 72bits。
            1.3：Type用来区分Frame Payload中的数据类型的。主要Type参见(注3)
            1.4：Flags用来做类型标记作用。不同的Type有不同的Flags值，表示不同的意义。
            1.5：R是保留位。该值必须设置为0。
            1.6：StreamIdentifier本Frame所属的流ID。当客户端和服务端建立TCP连接时，会先发送一个StreamID = 0的流做一些初始化工作。之后客户端和服务端从1号流开始发送请求响应。
            1.7：一个Frame由两大部分组成。上面72bits属于Frame Header下面为Frame Payload。
            
        2：对HTTP 1.x的兼容
            2.1：原来HTTP 1.x的HTTP header以及HTTP body都会存储在Frame Payload中，仅仅将Frame Header的Type标示一下这是HTTP 1.x格式。
            
        3：HPACK 压缩原理
            3.1:Frame Header使用 HPACK 进行压缩时，会先使用静态表(注1)进行转义压缩，还会使用动态表(注2)进行消息简化。
            3.2 HPACK 对HTTP/2所有头帧进行处理，包括 HEADERS，CONTINUATION, PUSH_PROMISE 三种头帧均受其约束。
            3.3 因为动态表的存在，在第一次请求之后，大多数的头会被标示为动态表或者静态表中的整数索引，大幅度缩小重复的头信息。
            3.4 因为 HPACK 支持霍夫曼编码，该编码为固定编码表，不容易受到基于压缩的攻击方式（例如CRIME）的攻击。
            
        4：Multipexing 多路复用原理
            4.1 Frame格式中有一个流ID，我们是使用它进行多路复用的：多个流可能被分配在一个TCP连接中，在分发处理的时候通过FrameID进行分发。
            4.2 每次请求/响应我们使用不同的Frame ID，这样我们就可以在一个TCP连接中通讯大量的Frame了。（注4）
            
        5：Server-Push 服务器推送 
            5.1 该功能是SPDY协议首先引入，它的目的是服务器允许主动向客户端进行推送行为。（注5）
            
        6：Priority 流优先级
            6.1 该功能允许客户端指定流的优先级，但这个功能是非强制的，服务端可以忽略它。
            6.2 流的优先级可以在流创建时进行设置，过程中可以使用 PRIORITY 帧进行动态修改。
            6.3 流的优先级可以赋予1-256之间的权重。
            6.4 流可以指定流之间的依赖性，该依赖性会破坏流优先级的依赖关系。
            
        7：数据流量控制
            7.1 HTTP/2 每个流都有自己的流量窗口，它可以限制另一段发送的数据总量。
            7.2 如果数据量不足，发出端可以向接受端发送 WINDOW_UPDATE 帧消息来请求申请更多的空间来接受新数据。
            7.3 注意：只有 DATA 数据帧会受到流量控制。
            
        8：Setting 设置
            7.1 HTTP/2 框架允许设置一系列用于控制连接的设置，规范可参见(注6)
            7.2 每个设置是单向有效的，（从C->S，从S->C）设置可以不对称。
        
**补充**
1. [静态表](http2.github.io/http2-spec/compression.html#rfc.section.A)使用方式
    
    例如：
    
    我们要进行HTTP 1.x 的如下行为：
        :method=GET
        :path=/index.html
        
    我们查找静态表可以发现 :method=GET 对应索引值是2，:path=/index.html 对应索引值是5。
    于是我们给服务器发送一个Frame，该Frame的Payload部分存储 **0x0205**，Frame的Type设置为 **HTTPHeader** 类型，则表示
            
        该Frame为HTTP Header，请求内容为 GET /index.html
        
1. 动态表 使用方式
    
    例如：
    
    以我们常用的**User-Agent**为例，它的Key在静态表中索引值为**58**，但是其值太多变，是不存在静态表中的。
    
    1. 于是，我们第一次请求时，会将其Key使用**58**表示，但其值部分假设为 **Mozilla5.0(Windows NT 6.1;WOW64)** 会进行霍夫曼编码尝试压缩处理（若压缩后字符串长于压缩前，则不会采用压缩）。
    1. 服务端收到本请求后，会将这个**User-Agent**的值保存到一个动态表中，并对其值分配一个索引值，假设为**62**。
    1. 下次客户端再进行请求时，其**User-Agent**的值就可以直接发送该索引值**0x3E**表示**Mozilla5.0(Windows NT 6.1;WOW64)** 了。
    1. 动态表一般默认大小为 **4k** ,但该值可以在 SETTTINGS 框架中进行调整。

1. 主要TYPE值与定义

        Type            Code    Commit
        DATA            0x0     在请求或回应中携带的数据（常用基本帧类型）
        HEADERS         0x1     用来打开一个流，它包含一个请求或回应的头数据（常用基本帧类型）
        PRIORITY        0x2     设置流的优先级别
        RST_STREAM      0x3     强制中断一个流。这仅用于客户端或服务端单向决定中断流时使用，正常中断流不应该使用该方式。
        SETTINGS        0x4     建立HTTP/2连接时进行连接设置
        PUSH_PROMISE    0x5     由服务端向客户端发送的推送响应
        PING            0x6     向远程终端发送ping
        GOAWAY          0x7     通知远程终端，即将关闭本连接
        WINDOW_UPDATE   0x8     更新流控制窗口
        CONTINUATION    0x9     当单个HEADERS头过大时，用本消息发送附加HEADERS头信息
        
1. 关于流
    
    1. 在HTTP/1.1中，请求每次仅允许处理一个。客户端向服务端发送请求，服务器进行响应。客户端收到响应后，它才可以向服务器发送另一个请求。
    1. 在HTTP/2中，客户端可以同时创建多个请求，并且允许指定请求的回应顺序。
    1. 由客户端启动的流必须使用奇数的流ID，由服务端发起的流（例如ServerPush）必须使用偶数的流ID。
    1. 流可以通过客户端的HEADERS帧发起，可以通过服务端的PUSH_PROMISE帧发起。
    1. 通常情况下，客户端和服务端使用多少个活动的并发流是没有数量限制的，但是服务端可以通过SETTINGS来限制单个客户端允许使用的流数量。

1. 关于Server-Push

    1. 例如：客户端向服务端请求获取 /index.html，服务端知道 /index.html 中有对 /a.png 的引用，则服务端可以主动向客户端推送 a.png，而不是等待客户端解析后再请求。
    1. 一个Server-Push过程，是由服务端主动使用 PUSH_PROMISE 帧打开流启动的。
    1. 其过程细节和普通的请求/响应流程相同。
    1. 一旦客户端发现服务器推送的资源已经存在（例如在缓存中已有），客户端可以向服务端发送 RST_STREAM 帧以取消推送。
    
1. 常见Setting设置

        SETTINGS_HEADER_TABLE_SIZE      HPACKB标头表的最大大小                  默认值4096
        SETTINGS_ENABLE_PUSH            是否允许服务端主动Push给客户端          默认值true
        SETTINGS_MAX_CONCURRENT_STREAMS 允许打开的并发流的最大数量              默认值无限
        SETTINGS_INITIAL_WINDOW_SIZE    用于流控制的初始窗口大小                默认值65535
        SETTINGS_MAX_FRAME_SIZE         远程端点准备接受的最大帧大小            默认值16384
        SETTINGS_MAX_HEADER_LIST_SIZE   建议远程端点接受的头列表最大大小        默认值无限

### 一个简单的请求/响应流程

【1】请求由客户端发送 HEADERS 帧，打开一个流开始。
    
* 这个 HEADERS 帧中包含常见的HTTP请求标头，包括：
    * :method     请求方法，Get或Post
    * :path       请求的路径
    * :scheme     请求方案，通常是http或https
    * :authority  和Http1.1标头类似，它包含目标URL的权限部分      
* 这些请求标头在第一个帧中必须存在。在第一个 HEADERS 帧后，你可以继续发送包含任何数量的请求头。
* 若再发送的请求头的数量超过了最大帧大小，则客户端可以立即发送带有 CONTINUATION 标示的帧。
* 在发送的请求头的最后一帧，需要设置 END_HEADERS 标示，告诉远端没有更多的请求头了。
* 请求可以不包含数据（例如Get类型的请求），那么需要在初始 HEADERS 帧设置 END_HEADERS 标示。
* 请求可以包含数据（例如Post类型的请求），那么需要在 HEADERS 帧之后，发送任意数量的 DATA 帧，在最后一个帧需要设置 END_STREAM 标示。

【2】服务端准备发送响应时，和客户端的操作基本类似。

* 区别是服务端发送 HEADERS 帧中不需要客户端的HTTP请求标头，只需要

        :status     响应状态代码          
* 同样，服务端可以发一个 HEADERS 帧，或者在 HEADERS 帧后跟更多的 DATA 帧，使用 END_STREAM 标示全部完毕。
    
【补】HTTP/1.x消息格式 和 HTTP/2帧格式 转换对比

    +---------------------------+           +---------------------------+
    POST /resource HTTP/1.1                 HEADERS 
    Host: example.org                           - END_STREAM
    Content-Type: image/jpeg        ===>        - END_HEADERS
    Content-Length: 123                             :method = POST
    {Binary data} ...                               :path = /resource
    +---------------------------+                   :scheme = https
                                            +---------------------------+
                                            +---------------------------+
                                            CONTINUATION
                                                - END_HEADERS
                                                    :content-type = image/jpeg
                                                    :host = example.org
                                                    :content-length = 123
                                            +---------------------------+
                                            +---------------------------+
                                            DATA
                                                - END_STREAM
                                                {Binary data} ...
                                            +---------------------------+ 

### 简单性能测试

* 你可以访问[AakamaiDemo](https://http2.akamai.com/demo)测试了解其性能。
* 你可以下载[LoadImpact](https://loadimpact.com/http2old)进行对比测试。

### 使用与调试

* 根据[CanIUse](caniuse.com/#search=HTTP%2F2)统计，浏览器部分支持HTTP/2的包括Chrome 41+, FireFox 36+, Safari 9+, Windows10上的IE11和Edge.
* 你可以从[这里](https://github.com/http2/http2-spec/wiki/Implementations)获取最新的各种语言的HTTP/2服务库。
* 在Chrome中输入 ***chrome://net-internals/#http2*** 可以打开Chrome自带的HTTP/2查看工具，可以很方便的查看HTTP/2帧信息。
* 在Chrome中按下F12，进入开发者工具中可以查看Network部分，以获取帧信息。
* 你可以使用**curl** 这样的工具来调试协议，也可以使用**Wireshark** 查看HTTP/2帧信息。

### 参考书籍

1. [HTTP2讲解](https://yellow.gitbooks.io/http2-explained/content) 【中文】
1. [HTTPBis小组邮件列表和归档](http://lists.w3.org/Archives/Public/ietf-http-wg/)
1. [HTML版本的http2协议规范本体](https://httpwg.github.io/specs/rfc7540.html)
1. [Firefox http2网络细节](https://wiki.mozilla.org/Networking/http2)
1. [curl http2实现细节](http://curl.haxx.se/docs/http2.html)
1. [http2官方网站](http://http2.github.io/)
1. [http2的FAQ](http://http2.github.io/faq/)
1. [High Performance Browser Networking](http://chimera.labs.oreilly.com/books/1230000000545/ch12.html)

## MicroService 微服务架构

### 概述

微服务是一种架构理念。目的是有效的拆分应用功能，实现业务解耦，适合敏捷开发和部署。它具备以下特点：

* 一系列的独立的服务，共同组成一个系统。
* 每个服务有独立进程，可以独立部署。
* 每个服务拥有独立的业务。
* 可以分布式的管理。
### 优点

* 每个子服务都可以独立于其他服务执行，因此它具有更好的服务边界。
* 每个子服务代码更加容易理解和维护。
* 每个子服务可以选择最适合实现业务逻辑的技术，进行新技术更替时更为便捷。
* 每个子服务可以独立部署，为该业务准备特定的硬件设备。
* 每个子服务可以独立扩容，可以针对该服务进行扩容更多的服务组。
* 系统容错力上升，当单一子服务崩溃，其他子服务依然可以独立处理用户请求。
* 适合多人协同开发，降低沟通成本。
* 减少编译时间，减少启动消耗。
* 每个服务允许用不同的语言进行开发。

### 缺点

* 当子服务之间有配合需求时，保证其数据一致性非常痛苦。
* 进行跨子服务的集成测试代价很大。
* 基于性能考虑，各子服务提供的API接口应当是粗粒度的，有高灵活性的API。而这种API使用难度较高。
* 规范各子服务的沟通协议代价也很大。
* 服务间有额外的通信成本。
* 有更多的重复性工作。