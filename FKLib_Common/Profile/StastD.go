//---------------------------------------------
package Profile
//---------------------------------------------
import (
	FMT "fmt"
	LOG "log"
	OS "os"
	RUNTIME "runtime"
	TIME "time"

	G2S "github.com/peterbourgon/g2s"
)
//---------------------------------------------
const (
	ENV_STATSD          = "STATSD_HOST"
	DEFAULT_STATSD_HOST = "172.17.42.1:8125"
	COLLECT_DELAY       = 1 * TIME.Minute
	SERVICE             = "[STATSD_PPROF]"
)
//---------------------------------------------
var (
	_statter G2S.Statter
)
//---------------------------------------------
func init() {
	addr := DEFAULT_STATSD_HOST
	if env := OS.Getenv(ENV_STATSD); env != "" {
		addr = env
	}

	s, err := G2S.Dial("udp", addr)
	if err == nil {
		_statter = s
	} else {
		_statter = G2S.Noop()
		LOG.Println(err)
	}

	go pprof_task()
}
//---------------------------------------------
// profiling task
func pprof_task() {
	for {
		<-TIME.After(COLLECT_DELAY)
		collect()
	}
}
//---------------------------------------------
// collect & publish to statsd
func collect() {
	var tag string
	hostname, err := OS.Hostname()
	if err != nil {
		LOG.Println(SERVICE, err)
		return
	}
	tag = hostname + ".pprof"

	// collect
	memstats := &RUNTIME.MemStats{}
	RUNTIME.ReadMemStats(memstats)

	_statter.Gauge(1.0, tag+".NumGoroutine", FMT.Sprint(RUNTIME.NumGoroutine()))
	_statter.Gauge(1.0, tag+".NumCgoCall", FMT.Sprint(RUNTIME.NumCgoCall()))
	if memstats.NumGC > 0 {
		_statter.Timing(1.0, tag+".PauseTotal", TIME.Duration(memstats.PauseTotalNs))
		_statter.Timing(1.0, tag+".LastPause", TIME.Duration(memstats.PauseNs[(memstats.NumGC+255)%256]))
		_statter.Gauge(1.0, tag+".NumGC", FMT.Sprint(memstats.NumGC))
		_statter.Gauge(1.0, tag+".Alloc", FMT.Sprint(memstats.Alloc))
		_statter.Gauge(1.0, tag+".TotalAlloc", FMT.Sprint(memstats.TotalAlloc))
		_statter.Gauge(1.0, tag+".Sys", FMT.Sprint(memstats.Sys))
		_statter.Gauge(1.0, tag+".Lookups", FMT.Sprint(memstats.Lookups))
		_statter.Gauge(1.0, tag+".Mallocs", FMT.Sprint(memstats.Mallocs))
		_statter.Gauge(1.0, tag+".Frees", FMT.Sprint(memstats.Frees))
		_statter.Gauge(1.0, tag+".HeapAlloc", FMT.Sprint(memstats.HeapAlloc))
		_statter.Gauge(1.0, tag+".HeapSys", FMT.Sprint(memstats.HeapSys))
		_statter.Gauge(1.0, tag+".HeapIdle", FMT.Sprint(memstats.HeapIdle))
		_statter.Gauge(1.0, tag+".HeapInuse", FMT.Sprint(memstats.HeapInuse))
		_statter.Gauge(1.0, tag+".HeapReleased", FMT.Sprint(memstats.HeapReleased))
		_statter.Gauge(1.0, tag+".HeapObjects", FMT.Sprint(memstats.HeapObjects))
		_statter.Gauge(1.0, tag+".StackInuse", FMT.Sprint(memstats.StackInuse))
		_statter.Gauge(1.0, tag+".StackSys", FMT.Sprint(memstats.StackSys))
		_statter.Gauge(1.0, tag+".MSpanInuse", FMT.Sprint(memstats.MSpanInuse))
		_statter.Gauge(1.0, tag+".MSpanSys", FMT.Sprint(memstats.MSpanSys))
		_statter.Gauge(1.0, tag+".MCacheInuse", FMT.Sprint(memstats.MCacheInuse))
		_statter.Gauge(1.0, tag+".MCacheSys", FMT.Sprint(memstats.MCacheSys))
		_statter.Gauge(1.0, tag+".BuckHashSys", FMT.Sprint(memstats.BuckHashSys))
		_statter.Gauge(1.0, tag+".GCSys", FMT.Sprint(memstats.GCSys))
		_statter.Gauge(1.0, tag+".OtherSys", FMT.Sprint(memstats.OtherSys))
	}
}
//---------------------------------------------