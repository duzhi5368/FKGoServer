//---------------------------------------------
package utils

//---------------------------------------------

import (
	RUNTIME "runtime"

	LOG "github.com/Sirupsen/logrus"
	SPEW "github.com/davecgh/go-spew/spew"
)

//---------------------------------------------
// 产生panic时的调用栈打印
func PrintPanicStack(extras ...interface{}) {
	if x := recover(); x != nil {
		LOG.Error(x)
		i := 0
		funcName, file, line, ok := RUNTIME.Caller(i)
		for ok {
			LOG.Errorf("frame %v:[func:%v,file:%v,line:%v]\n", i, RUNTIME.FuncForPC(funcName).Name(), file, line)
			i++
			funcName, file, line, ok = RUNTIME.Caller(i)
		}

		for k := range extras {
			LOG.Errorf("EXRAS#%v DATA:%v\n", k, SPEW.Sdump(extras[k]))
		}
	}
}
