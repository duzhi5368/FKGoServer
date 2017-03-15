//---------------------------------------------
package framework

//---------------------------------------------
import (
	PROTO "FKGoServer/FKGRpc_Snowflake/Proto"
	ETCDCLIENT "FKGoServer/FKLib_Common/ETCDClient"
	ERRORS "errors"
	FMT "fmt"
	RAND "math/rand"
	OS "os"
	STRCONV "strconv"
	TIME "time"

	LOG "github.com/Sirupsen/logrus"
	ETCD "github.com/coreos/etcd/client"
	CONTEXT "golang.org/x/net/context"
)

//---------------------------------------------
const (
	SERVICE        = "[SNOWFLAKE]"
	ENV_MACHINE_ID = "MACHINE_ID" // specific machine id
	PATH           = "/seqs/"
	UUID_KEY       = "/seqs/snowflake-uuid"
	BACKOFF        = 100  // max backoff delay millisecond
	CONCURRENT     = 128  // max concurrent connections to etcd
	UUID_QUEUE     = 1024 // uuid process queue
)

//---------------------------------------------
const (
	TS_MASK         = 0x1FFFFFFFFFF // 41bit
	SN_MASK         = 0xFFF         // 12bit
	MACHINE_ID_MASK = 0x3FF         // 10bit
)

type Server struct {
	machine_id  uint64 // 10-bit machine id
	client_pool chan ETCD.KeysAPI
	ch_proc     chan chan uint64
}

//---------------------------------------------
// 进行服务初始化行为
func (s *Server) Func_Init() {
	s.client_pool = make(chan ETCD.KeysAPI, CONCURRENT)
	s.ch_proc = make(chan chan uint64, UUID_QUEUE)

	// 初始化客户端池
	for i := 0; i < CONCURRENT; i++ {
		s.client_pool <- ETCDCLIENT.KeysAPI()
	}

	// 检查是否用户机器ID已设置
	if env := OS.Getenv(ENV_MACHINE_ID); env != "" {
		if id, err := STRCONV.Atoi(env); err == nil {
			s.machine_id = (uint64(id) & MACHINE_ID_MASK) << 12
			LOG.Info("机器ID被指定:", id)
		} else {
			LOG.Panic(err)
			OS.Exit(-1)
		}
	} else {
		// 若没有被设置，则进行初始化生成
		s.func_InitMachineID()
	}

	// 创建协程生成UUID
	go s.func_UUIDCreator()
}

//---------------------------------------------
func (s *Server) func_InitMachineID() {
	client := <-s.client_pool
	defer func() { s.client_pool <- client }()

	for {
		// 获取UUID键值
		resp, err := client.Get(CONTEXT.Background(), UUID_KEY, nil)
		if err != nil {
			LOG.Panic(err)
			OS.Exit(-1)
		}

		prevValue, err := STRCONV.Atoi(resp.Node.Value)
		if err != nil {
			LOG.Panic(err)
			OS.Exit(-1)
		}
		prevIndex := resp.Node.ModifiedIndex

		// 比较并交换
		resp, err = client.Set(CONTEXT.Background(), UUID_KEY, FMT.Sprint(prevValue+1), &ETCD.SetOptions{PrevIndex: prevIndex})
		if err != nil {
			func_RandomDelay()
			continue
		}

		// record serial number of this service, already shifted
		s.machine_id = (uint64(prevValue+1) & MACHINE_ID_MASK) << 12
		return
	}
}

//---------------------------------------------
// 获取一个Key的下一个value,类似于mysql的自叠加
func (s *Server) Next(ctx CONTEXT.Context, in *PROTO.Snowflake_Key) (*PROTO.Snowflake_Value, error) {
	client := <-s.client_pool
	defer func() { s.client_pool <- client }()
	key := PATH + in.Name
	for {
		// 获取Key
		resp, err := client.Get(CONTEXT.Background(), key, nil)
		if err != nil {
			LOG.Error(err)
			return nil, ERRORS.New("Key not exists, need to create first")
		}

		prevValue, err := STRCONV.Atoi(resp.Node.Value)
		if err != nil {
			LOG.Error(err)
			return nil, ERRORS.New("marlformed value")
		}
		prevIndex := resp.Node.ModifiedIndex

		// 比较并交换
		resp, err = client.Set(CONTEXT.Background(), key, FMT.Sprint(prevValue+1), &ETCD.SetOptions{PrevIndex: prevIndex})
		if err != nil {
			func_RandomDelay()
			continue
		}
		return &PROTO.Snowflake_Value{int64(prevValue + 1)}, nil
	}
}

//---------------------------------------------
// 生成唯一UUID
func (s *Server) GetUUID(CONTEXT.Context, *PROTO.Snowflake_NullRequest) (*PROTO.Snowflake_UUID, error) {
	req := make(chan uint64, 1)
	s.ch_proc <- req
	return &PROTO.Snowflake_UUID{<-req}, nil
}

//---------------------------------------------
// UUID 生成器
func (s *Server) func_UUIDCreator() {
	var sn uint64     // 12位的序列号
	var last_ts int64 // 最后的时间戳
	for {
		ret := <-s.ch_proc
		// 开始计算序列码
		t := func_GetTimeStamp()
		if t < last_ts { // clock shift backward
			LOG.Error("clock shift happened, waiting until the clock moving to the next millisecond.")
			t = s.func_WaitUtil(last_ts)
		}

		if last_ts == t { // 同一毫秒
			sn = (sn + 1) & SN_MASK
			if sn == 0 { // 序列码溢出，等待下一毫秒
				t = s.func_WaitUtil(last_ts)
			}
		} else { // 新一毫秒，重置序列码为0
			sn = 0
		}
		// 记录最后的时间戳
		last_ts = t

		// 生成UUID格式为
		// 0		0.................0		0..............0	0........0
		// 1-bit 无用	41bit 时间戳			10bit 机器ID		12bit 序列号
		var uuid uint64
		uuid |= (uint64(t) & TS_MASK) << 22
		uuid |= s.machine_id
		uuid |= sn
		ret <- uuid
	}
}

//---------------------------------------------
// 持续等待到指定时间
func (s *Server) func_WaitUtil(last_ts int64) int64 {
	t := func_GetTimeStamp()
	for t <= last_ts {
		t = func_GetTimeStamp()
	}
	return t
}

//---------------------------------------------
// 进行随机延迟
func func_RandomDelay() {
	<-TIME.After(TIME.Duration(RAND.Int63n(BACKOFF)) * TIME.Millisecond)
}

//---------------------------------------------
// 获取时间戳
func func_GetTimeStamp() int64 {
	return TIME.Now().UnixNano() / int64(TIME.Millisecond)
}

//---------------------------------------------
