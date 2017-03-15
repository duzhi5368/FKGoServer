//---------------------------------------------
package packet

//---------------------------------------------
import (
	LOG "log"
	REFLECT "reflect"
)

//---------------------------------------------
type FastPack interface {
	Pack(w *Packet)
}

//---------------------------------------------
// 进行数据封包
func Func_Pack(tos int16, tbl interface{}, writer *Packet) []byte {
	// 创建包Writer对象
	if writer == nil {
		writer = Writer()
	}

	// 写入消息编号
	writer.WriteS16(tos)

	// 检查是否有数据表
	if tbl == nil {
		return writer.Data()
	}

	// 进行快速打包
	if fastpack, ok := tbl.(FastPack); ok {
		fastpack.Pack(writer)
		return writer.Data()
	}

	// 反射打包
	func_Pack(REFLECT.ValueOf(tbl), writer)

	// 返回字节数据
	return writer.Data()
}

//---------------------------------------------
func func_Pack(v REFLECT.Value, writer *Packet) {
	switch v.Kind() {
	case REFLECT.Bool:
		writer.WriteBool(v.Bool())
	case REFLECT.Uint8:
		writer.WriteByte(byte(v.Uint()))
	case REFLECT.Uint16:
		writer.WriteU16(uint16(v.Uint()))
	case REFLECT.Uint32:
		writer.WriteU32(uint32(v.Uint()))
	case REFLECT.Uint64:
		writer.WriteU64(uint64(v.Uint()))

	case REFLECT.Int16:
		writer.WriteS16(int16(v.Int()))
	case REFLECT.Int32:
		writer.WriteS32(int32(v.Int()))
	case REFLECT.Int64:
		writer.WriteS64(int64(v.Int()))

	case REFLECT.Float32:
		writer.WriteFloat32(float32(v.Float()))
	case REFLECT.Float64:
		writer.WriteFloat64(float64(v.Float()))

	case REFLECT.String:
		writer.WriteString(v.String())
	case REFLECT.Ptr, REFLECT.Interface:
		if v.IsNil() {
			return
		}
		func_Pack(v.Elem(), writer)
	case REFLECT.Slice:
		if bs, ok := v.Interface().([]byte); ok { // special treat for []bytes
			writer.WriteBytes(bs)
		} else {
			l := v.Len()
			writer.WriteU16(uint16(l))
			for i := 0; i < l; i++ {
				func_Pack(v.Index(i), writer)
			}
		}
	case REFLECT.Struct:
		numFields := v.NumField()
		for i := 0; i < numFields; i++ {
			func_Pack(v.Field(i), writer)
		}
	default:
		LOG.Println("不可识别的包类型:", v)
	}
}

//---------------------------------------------
