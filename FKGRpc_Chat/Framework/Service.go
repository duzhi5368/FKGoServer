//---------------------------------------------
package framework
//---------------------------------------------
import (
	JSON "encoding/json"
	ERRORS "errors"
	FMT "fmt"
	OS "os"
	SIGNAL "os/signal"
	STRCONV "strconv"
	SYNC "sync"
	ATOMIC "sync/atomic"
	SYSCALL "syscall"
	TIME "time"

	KAFKA "FKGoServer/FKGRpc_Chat/Kafka"

	SARAMA "github.com/Shopify/sarama"
	LOG "github.com/Sirupsen/logrus"
	BOLT "github.com/boltdb/bolt"
	CONTEXT "golang.org/x/net/context"

	PROTO "FKGoServer/FKGRpc_Chat/Proto"

	CLI "gopkg.in/urfave/cli.v2"
)
//---------------------------------------------
var (
	OK                   = &PROTO.Chat_Nil{}
	ERROR_ALREADY_EXISTS = ERRORS.New("id already exists")
	ERROR_NOT_EXISTS     = ERRORS.New("id not exists")
)

type Consumer struct {
	offset   int64 // next message offset
	pushFunc func(msg *PROTO.Chat_Message)
}

// endpoint definition
type EndPoint struct {
	retention   int
	StartOffset int64 // offset of the first message
	Inbox       []PROTO.Chat_Message
	consumers   map[uint64]*Consumer
	chReady     chan struct{}
	die         chan struct{}
	mu          SYNC.Mutex
}

func newEndPoint(retention int) *EndPoint {
	ep := &EndPoint{}
	ep.retention = retention
	ep.chReady = make(chan struct{}, 1)
	ep.consumers = make(map[uint64]*Consumer)
	ep.StartOffset = 1
	ep.die = make(chan struct{})
	go ep.pushTask()
	return ep
}

// push a message to this endpoint
func (ep *EndPoint) push(msg *PROTO.Chat_Message) {
	if len(ep.Inbox) > ep.retention {
		ep.Inbox = append(ep.Inbox[1:], *msg)
		ep.StartOffset++
	} else {
		ep.Inbox = append(ep.Inbox, *msg)
	}
	ep.notifyConsumers()
}

// closes this endpoint
func (ep *EndPoint) close() {
	close(ep.die)
}

func (ep *EndPoint) notifyConsumers() {
	select {
	case ep.chReady <- struct{}{}:
	default:
	}
}

func (ep *EndPoint) pushTask() {
	for {
		select {
		case <-ep.chReady:
			ep.mu.Lock()
			for _, consumer := range ep.consumers {
				idx := consumer.offset - ep.StartOffset
				if idx < 0 { // lag behind many
					idx = 0
				}
				for i := idx; i < int64(len(ep.Inbox)); i++ {
					ep.Inbox[i].Offset = i + ep.StartOffset
					consumer.pushFunc(&ep.Inbox[i])
				}
				consumer.offset = ep.StartOffset + int64(len(ep.Inbox))
			}
			ep.mu.Unlock()
		case <-ep.die:
		}
	}
}

// server definition
type Server struct {
	consumerid_autoinc uint64
	kafkaOffset        int64
	offsetBucket       string
	retention          int
	boltdb             string
	bucket             string
	interval           TIME.Duration
	eps                map[uint64]*EndPoint // end-point-s
	SYNC.RWMutex
}

func (s *Server) Func_Init(c *CLI.Context) {
	s.retention = c.Int("retention")
	s.boltdb = c.String("boltdb")
	s.bucket = c.String("bucket")
	s.interval = c.Duration("write-interval")
	s.offsetBucket = c.String("kafka-bucket")

	s.eps = make(map[uint64]*EndPoint)
	s.restore()
	go s.receive()
	go s.persistence_task()
}

func (s *Server) read_ep(id uint64) *EndPoint {
	s.RLock()
	defer s.RUnlock()
	return s.eps[id]
}

// subscribe to an endpoint & receive server streams
func (s *Server) Subscribe(p *PROTO.Chat_Consumer, stream PROTO.ChatService_SubscribeServer) error {
	ep := s.read_ep(p.Id)
	if ep == nil {
		LOG.Errorf("cannot find endpoint %v", p)
		return ERROR_NOT_EXISTS
	}

	consumerid := ATOMIC.AddUint64(&s.consumerid_autoinc, 1)
	e := make(chan error, 1)

	// activate consumer
	ep.mu.Lock()

	// from newest
	if p.From == -1 {
		p.From = ep.StartOffset + int64(len(ep.Inbox))
	}
	ep.consumers[consumerid] = &Consumer{p.From, func(msg *PROTO.Chat_Message) {
		if err := stream.Send(msg); err != nil {
			select {
			case e <- err:
			default:
			}
		}
	}}
	ep.mu.Unlock()
	defer func() {
		ep.mu.Lock()
		delete(ep.consumers, consumerid)
		ep.mu.Unlock()
	}()

	ep.notifyConsumers()

	select {
	case <-stream.Context().Done():
	case err := <-e:
		return err
	}
	return nil
}

func (s *Server) receive() {
	consumer, err := KAFKA.NewConsumer()
	if err != nil {
		LOG.Fatalln(err)
	}

	defer consumer.Close()
	partionConsumer, err := consumer.ConsumePartition(KAFKA.ChatTopic, 0, s.kafkaOffset)
	LOG.Info("kafkaOffset ", s.kafkaOffset)
	if err != nil {
		LOG.Fatalln(err)
	}
	defer partionConsumer.Close()
	for {
		select {
		case msg := <-partionConsumer.Messages():
			LOG.WithField("msg", msg).WithField("OFFSET", s.kafkaOffset).WithField("IsNew", s.kafkaOffset < msg.Offset).Info("Receive")
			if s.kafkaOffset < msg.Offset {
				chat := new(PROTO.Chat_Message)
				JSON.Unmarshal(msg.Key, &chat.Id)
				chat.Body = msg.Value
				ep := s.read_ep(chat.Id)
				s.Lock()
				if ep != nil {
					ep.mu.Lock()
					ep.push(chat)
					ep.mu.Unlock()
				}
				s.kafkaOffset = msg.Offset
				s.Unlock()
			}
		}
	}

}

func (s *Server) Reg(ctx CONTEXT.Context, p *PROTO.Chat_Id) (*PROTO.Chat_Nil, error) {
	s.Lock()
	defer s.Unlock()
	ep := s.eps[p.Id]
	if ep != nil {
		LOG.Errorf("id already exists:%v", p.Id)
		return nil, ERROR_ALREADY_EXISTS
	}

	s.eps[p.Id] = newEndPoint(s.retention)
	return OK, nil
}

func (s *Server) Query(ctx CONTEXT.Context, crange *PROTO.Chat_ConsumeRange) (*PROTO.Chat_List, error) {
	ep := s.read_ep(crange.Id)
	if ep == nil {
		return nil, ERROR_NOT_EXISTS
	}

	ep.mu.Lock()
	defer ep.mu.Unlock()

	if crange.From < ep.StartOffset {
		crange.From = ep.StartOffset
	}

	if crange.To > ep.StartOffset+int64(len(ep.Inbox))-1 {
		crange.To = ep.StartOffset + int64(len(ep.Inbox)) - 1
	}

	list := &PROTO.Chat_List{}
	if crange.To > crange.From {
		return list, nil
	}

	for i := crange.From; i <= crange.To; i++ {
		msg := ep.Inbox[i-ep.StartOffset]
		msg.Offset = i
		list.Messages = append(list.Messages, &msg)
	}

	return list, nil
}

func (s *Server) Latest(ctx CONTEXT.Context, crange *PROTO.Chat_ConsumeLatest) (*PROTO.Chat_List, error) {
	ep := s.read_ep(crange.Id)
	if ep == nil {
		return nil, ERROR_NOT_EXISTS
	}

	ep.mu.Lock()
	defer ep.mu.Unlock()

	list := &PROTO.Chat_List{}
	i := len(ep.Inbox) - int(crange.Length)
	if i < 0 {
		i = 0
	}
	for ; i < len(ep.Inbox); i++ {
		offset := int64(i) + ep.StartOffset
		msg := ep.Inbox[i]
		msg.Offset = offset
		list.Messages = append(list.Messages, &msg)
	}
	return list, nil
}

// persistence endpoints into db
func (s *Server) persistence_task() {
	ticker := TIME.NewTicker(s.interval)
	defer ticker.Stop()
	db := s.open_db()
	sig := make(chan OS.Signal, 1)
	SIGNAL.Notify(sig, SYSCALL.SIGTERM, SYSCALL.SIGINT)

	for {
		select {
		case <-ticker.C:
			s.dump(db)
		case nr := <-sig:
			s.dump(db)
			db.Close()
			LOG.Info(nr)
			OS.Exit(0)
		}
	}
}

func (s *Server) open_db() *BOLT.DB {
	db, err := BOLT.Open(s.boltdb, 0600, nil)
	if err != nil {
		LOG.Panic(err)
		OS.Exit(-1)
	}
	// create bulket
	db.Update(func(tx *BOLT.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(s.bucket))
		if err != nil {
			LOG.Panicf("create bucket: %s", err)
			OS.Exit(-1)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(s.offsetBucket))
		if err != nil {
			LOG.Panicf("create bucket: %s", err)
			OS.Exit(-1)
		}
		return nil
	})
	return db
}

func (s *Server) dump(db *BOLT.DB) {
	// save offset
	db.Update(func(tx *BOLT.Tx) error {
		s.Lock()
		// write kafka offset
		b := tx.Bucket([]byte(s.offsetBucket))
		bin, _ := JSON.Marshal(s.kafkaOffset)
		if err := b.Put([]byte(s.offsetBucket), bin); err != nil {
			LOG.Error(err)
		}

		// write endpoints
		b = tx.Bucket([]byte(s.bucket))
		eps := make(map[uint64]*EndPoint)
		for k, v := range s.eps {
			eps[k] = v
		}

		for k, ep := range eps {
			ep.mu.Lock()
			if bin, err := JSON.Marshal(ep); err != nil {
				LOG.Error("cannot marshal:", err)
			} else if err := b.Put([]byte(FMT.Sprint(k)), bin); err != nil {
				LOG.Error(err)
			}
			ep.mu.Unlock()
		}
		s.Unlock()
		return nil
	})
}

func (s *Server) restore() {
	// restore data from db file
	db := s.open_db()
	defer db.Close()
	count := 0
	db.View(func(tx *BOLT.Tx) error {
		b := tx.Bucket([]byte(s.offsetBucket))
		s.kafkaOffset = SARAMA.OffsetNewest
		b.ForEach(func(k, v []byte) error {
			JSON.Unmarshal(v, &s.kafkaOffset)
			return nil
		})

		b = tx.Bucket([]byte(s.bucket))
		b.ForEach(func(k, v []byte) error {
			ep := newEndPoint(s.retention)
			ep.mu.Lock()
			if err := JSON.Unmarshal(v, &ep); err != nil {
				LOG.Fatalln("chat data corrupted:", err)
			}

			id, err := STRCONV.ParseUint(string(k), 0, 64)
			if err != nil {
				LOG.Fatalln("chat data corrupted:", err)
			}

			// settings
			if len(ep.Inbox) > s.retention {
				remove := len(ep.Inbox) - s.retention
				if remove > 0 {
					ep.Inbox = ep.Inbox[remove:]
				}
			}
			s.eps[id] = ep
			count++
			ep.mu.Unlock()
			return nil
		})
		return nil
	})

	LOG.Infof("restored %v chats", count)
}
