//---------------------------------------------
package db

//---------------------------------------------
import (
	OS "os"
	TIME "time"

	LOG "github.com/Sirupsen/logrus"
	MGO "gopkg.in/mgo.v2"
)

//---------------------------------------------
type Database struct {
	session *MGO.Session
	latch   chan *MGO.Session
}

//---------------------------------------------
var (
	DefaultDatabase Database
)

//---------------------------------------------
// 对外接口
func Func_InitDB(mongodb string, concurrent int, timeout TIME.Duration) {
	DefaultDatabase.func_Init(mongodb, concurrent, timeout)
}

//---------------------------------------------
// 数据库初始化工作
func (db *Database) func_Init(addr string, concurrent int, timeout TIME.Duration) {
	// 创建Latch
	db.latch = make(chan *MGO.Session, concurrent)
	sess, err := MGO.Dial(addr)
	if err != nil {
		LOG.Println("mongodb连接失败 - ", addr, err)
		OS.Exit(-1)
	}

	// 设置数据库参数
	sess.SetMode(MGO.Strong, true)
	sess.SetSocketTimeout(timeout)
	sess.SetCursorTimeout(0)
	db.session = sess

	for k := 0; k < cap(db.latch); k++ {
		db.latch <- sess.Copy()
	}
}

//---------------------------------------------
// 执行
func (db *Database) Execute(f func(sess *MGO.Session) error) error {
	// latch control
	sess := <-db.latch
	defer func() {
		db.latch <- sess
	}()
	sess.Refresh()
	return f(sess)
}

//---------------------------------------------
