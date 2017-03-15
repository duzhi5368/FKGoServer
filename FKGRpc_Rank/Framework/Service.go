//---------------------------------------------
package Framework
//---------------------------------------------
import (
	ERRORS "errors"
	FMT "fmt"
	OS "os"
	SIGNAL "os/signal"
	STRCONV "strconv"
	SYNC "sync"
	SYSCALL "syscall"
	TIME "time"

	LOG "github.com/Sirupsen/logrus"
	BOLT "github.com/boltdb/bolt"
	CONTEXT "golang.org/x/net/context"
	PROTO "FKGoServer/FKGRpc_Rank/Proto"
	RANKSET "FKGoServer/FKGRpc_Rank/RankSet"
)
//---------------------------------------------
const (
	SERVICE = "[RANK]"
)
//---------------------------------------------
const (
	BOLTDB_FILE    = "/data/RANK-DUMP.DAT"
	BOLTDB_BUCKET  = "RANKING"
	CHANGES_SIZE   = 65536
	CHECK_INTERVAL = TIME.Minute // if ranking has changed, how long to check
)
//---------------------------------------------
var (
	OK                    = &PROTO.Ranking_Nil{}
	ERROR_NAME_NOT_EXISTS = ERRORS.New("name not exists")
)

type Server struct {
	ranks   map[uint64]*RANKSET.RankSet
	pending chan uint64
	SYNC.RWMutex
}

func (s *Server) Func_Init() {
	s.ranks = make(map[uint64]*RANKSET.RankSet)
	s.pending = make(chan uint64, CHANGES_SIZE)
	s.restore()
	go s.persistence_task()
}

func (s *Server) lock_read(f func()) {
	s.RLock()
	defer s.RUnlock()
	f()
}

func (s *Server) lock_write(f func()) {
	s.Lock()
	defer s.Unlock()
	f()
}

func (s *Server) RankChange(ctx CONTEXT.Context, p *PROTO.Ranking_Change) (*PROTO.Ranking_Nil, error) {
	// check name existence
	var rs *RANKSET.RankSet
	s.lock_write(func() {
		rs = s.ranks[p.SetId]
		if rs == nil {
			rs = RANKSET.NewRankSet()
			s.ranks[p.SetId] = rs
		}
	})

	// apply update on the rankset
	rs.Update(p.UserId, p.Score)
	s.pending <- p.SetId
	return OK, nil
}

func (s *Server) QueryRankRange(ctx CONTEXT.Context, p *PROTO.Ranking_Range) (*PROTO.Ranking_RankList, error) {
	var rs *RANKSET.RankSet
	s.lock_read(func() {
		rs = s.ranks[p.SetId]
	})

	if rs == nil {
		return nil, ERROR_NAME_NOT_EXISTS
	}

	ids, cups := rs.GetList(int(p.A), int(p.B))
	return &PROTO.Ranking_RankList{UserIds: ids, Scores: cups}, nil
}

func (s *Server) QueryUsers(ctx CONTEXT.Context, p *PROTO.Ranking_Users) (*PROTO.Ranking_UserList, error) {
	var rs *RANKSET.RankSet
	s.lock_read(func() {
		rs = s.ranks[p.SetId]
	})

	if rs == nil {
		return nil, ERROR_NAME_NOT_EXISTS
	}

	ranks := make([]int32, 0, len(p.UserIds))
	scores := make([]int32, 0, len(p.UserIds))
	for _, id := range p.UserIds {
		rank, score := rs.Rank(id)
		ranks = append(ranks, rank)
		scores = append(scores, score)
	}
	return &PROTO.Ranking_UserList{Ranks: ranks, Scores: scores}, nil
}

func (s *Server) DeleteSet(ctx CONTEXT.Context, p *PROTO.Ranking_SetId) (*PROTO.Ranking_Nil, error) {
	s.lock_write(func() {
		delete(s.ranks, p.SetId)
	})
	return OK, nil
}

func (s *Server) DeleteUser(ctx CONTEXT.Context, p *PROTO.Ranking_DeleteUserRequest) (*PROTO.Ranking_Nil, error) {
	var rs *RANKSET.RankSet
	s.lock_read(func() {
		rs = s.ranks[p.SetId]
	})
	if rs == nil {
		return nil, ERROR_NAME_NOT_EXISTS
	}
	rs.Delete(p.UserId)
	return OK, nil
}

// persistence ranking tree into db
func (s *Server) persistence_task() {
	timer := TIME.After(CHECK_INTERVAL)
	db := s.open_db()
	changes := make(map[uint64]bool)
	sig := make(chan OS.Signal, 1)
	SIGNAL.Notify(sig, SYSCALL.SIGTERM, SYSCALL.SIGINT)

	for {
		select {
		case key := <-s.pending:
			changes[key] = true
		case <-timer:
			s.dump(db, changes)
			if len(changes) > 0 {
				LOG.Infof("perisisted %v rankset:", len(changes))
			}
			changes = make(map[uint64]bool)
			timer = TIME.After(CHECK_INTERVAL)
		case nr := <-sig:
			s.dump(db, changes)
			db.Close()
			LOG.Info(nr)
			OS.Exit(0)
		}
	}
}

func (s *Server) open_db() *BOLT.DB {
	db, err := BOLT.Open(BOLTDB_FILE, 0600, nil)
	if err != nil {
		LOG.Panic(err)
		OS.Exit(-1)
	}
	// create bulket
	db.Update(func(tx *BOLT.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(BOLTDB_BUCKET))
		if err != nil {
			LOG.Panicf("create bucket: %s", err)
			OS.Exit(-1)
		}
		return nil
	})
	return db
}

func (s *Server) dump(db *BOLT.DB, changes map[uint64]bool) {
	db.Update(func(tx *BOLT.Tx) error {
		b := tx.Bucket([]byte(BOLTDB_BUCKET))
		for k := range changes {
			// marshal
			var rs *RANKSET.RankSet
			s.lock_read(func() {
				rs = s.ranks[k]
			})

			if rs == nil { // rankset deletion
				b.Delete([]byte(FMT.Sprint(k)))
			} else { // serialization and save
				bin, err := rs.Marshal()
				if err != nil {
					LOG.Error(err)
					continue
				}
				b.Put([]byte(FMT.Sprint(k)), bin)
			}
		}
		return nil
	})
}

func (s *Server) restore() {
	// restore data from db file
	db := s.open_db()
	defer db.Close()
	db.View(func(tx *BOLT.Tx) error {
		b := tx.Bucket([]byte(BOLTDB_BUCKET))
		b.ForEach(func(k, v []byte) error {
			rs := RANKSET.NewRankSet()
			err := rs.Unmarshal(v)
			if err != nil {
				LOG.Panic("rank data corrupted:", err)
				OS.Exit(-1)
			}
			id, err := STRCONV.ParseUint(string(k), 0, 64)
			if err != nil {
				LOG.Panic("rank data corrupted:", err)
				OS.Exit(-1)
			}
			s.ranks[id] = rs
			return nil
		})
		return nil
	})
}