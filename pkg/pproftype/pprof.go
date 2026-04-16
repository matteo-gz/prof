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

type FamilyType string

const (
	FamilySampling FamilyType = "sampling"
	FamilyTrace    FamilyType = "trace"
	FamilySnapshot FamilyType = "snapshot"
	FamilyMeta     FamilyType = "meta"
)

func Family(name string) FamilyType {
	switch name {
	case Profile:
		return FamilySampling
	case Trace:
		return FamilyTrace
	case Cmdline, Symbol:
		return FamilyMeta
	default:
		return FamilySnapshot
	}
}

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

	AllSelectableTypes = []string{
		Profile,
		Trace,
		Goroutine,
		Heap,
		Allocs,
		Block,
		Mutex,
		Threadcreate,
	}
)

func IsValidType(name string) bool {
	for _, v := range List {
		if v == name {
			return true
		}
	}
	return false
}
