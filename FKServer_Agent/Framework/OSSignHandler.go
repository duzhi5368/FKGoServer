//---------------------------------------------
package framework

//---------------------------------------------
import (
	OS "os"
	SIGNAL "os/signal"
	SYSCALL "syscall"

	UTILS "FKGoServer/FKLib_Common/Utils"

	LOG "github.com/Sirupsen/logrus"
)

//---------------------------------------------
// 处理Unix内部信号
func func_HandlerUnixSign() {
	// 退出前必须产生panic时的调用栈打印
	defer UTILS.Func_PrintPanicStack()

	// 接收系统信号，填充至channel中
	ch := make(chan OS.Signal, 1)
	// 仅将系统的SIGTERM转发给channel，其他消息抛弃
	SIGNAL.Notify(ch, SYSCALL.SIGTERM)
	// 解析channel
	for {
		msg := <-ch // 将channel中数据填充至msg中
		switch msg {
		case SYSCALL.SIGTERM: // 如果收到了SIGTERM消息，则关闭Anget
			close(DIE_SIGN)
			LOG.Info("收到Unix关闭信号")
			LOG.Info("请等待Agent关闭...")
			SYNC_WAITGROUP.Wait() // 阻塞主线程的进行，等待全部协程执行完毕
			LOG.Info("Agent已关闭")
			OS.Exit(0)
		}
	}
}

//---------------------------------------------
