package tool

import "github.com/matteo-gz/prof/pkg/pproftype"

type Tool interface {
	Command() string
	Arg() arg
}
type arg interface {
	setName(name string)
	SetPort(port int)
	SetFile(file string)
	AppendEnv([]string) []string
	build() []string
}

type commonArg struct {
	name string
	port int
	file string
}

func (ca *commonArg) setName(name string) {
	ca.name = name
}

func (ca *commonArg) SetFile(file string) {
	ca.file = file
}
func (ca *commonArg) SetPort(port int) {
	ca.port = port
}

const (
	Sep     = " "
	HTTPTpl = "-http=0.0.0.0:%d"
)

func NewTool(name string) Tool {
	if name == pproftype.ExtPprof {
		return newPprofX()
	}
	return newTraceX()
}
