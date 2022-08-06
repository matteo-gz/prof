package pproftype

const (
	Cmdline      = "cmdline"
	Profile      = "profile"
	Symbol       = "symbol"
	Trace        = "trace"
	Allocs       = "allocs"
	Block        = "block"
	Goroutine    = "goroutine"
	Heap         = "heap"
	Mutex        = "mutex"
	Threadcreate = "threadcreate"

	ExtPprof   = "pprof"
	ExtTrace   = "trace"
	ExtTxt     = "txt"
	ExtUnknown = "unknown"
)

var (
	List = []string{
		Cmdline,
		Profile,
		Symbol,
		Trace,
		Allocs,
		Block,
		Goroutine,
		Heap,
		Mutex,
		Threadcreate,
	}
)
