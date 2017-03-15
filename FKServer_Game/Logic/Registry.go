//---------------------------------------------
package Logic

//---------------------------------------------
import (
	SYNC "sync"
)

//---------------------------------------------
type Registry struct {
	records map[int32]interface{} // id -> v
	SYNC.RWMutex
}

//---------------------------------------------
var (
	_default_registry Registry
)

//---------------------------------------------
func init() {
	_default_registry.init()
}

//---------------------------------------------
func (r *Registry) init() {
	r.records = make(map[int32]interface{})
}

//---------------------------------------------
// 注册用户
func (r *Registry) Register(id int32, v interface{}) {
	r.Lock()
	r.records[id] = v
	r.Unlock()
}

//---------------------------------------------
// 移除用户
func (r *Registry) Unregister(id int32) {
	r.Lock()
	delete(r.records, id)
	r.Unlock()
}

//---------------------------------------------
// 查询用户是否存在
func (r *Registry) Query(id int32) (x interface{}) {
	r.RLock()
	x = r.records[id]
	r.RUnlock()
	return
}

//---------------------------------------------
// 查询当前在线用户个数
func (r *Registry) Count() (count int) {
	r.RLock()
	count = len(r.records)
	r.RUnlock()
	return
}

//---------------------------------------------
func Register(id int32, v interface{}) {
	_default_registry.Register(id, v)
}

//---------------------------------------------
func Unregister(id int32) {
	_default_registry.Unregister(id)
}

//---------------------------------------------
func Query(id int32) interface{} {
	return _default_registry.Query(id)
}

//---------------------------------------------
func Count() int {
	return _default_registry.Count()
}

//---------------------------------------------
