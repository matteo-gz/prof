package tool

import (
	"fmt"
	"strings"
)

type pprofX struct {
	name string
	arg  *ArgPprofX
}

func (p pprofX) Command() string {
	return strings.Join(p.arg.build(), Sep)
}
func (p pprofX) Arg() arg {
	return p.arg
}
func newPprofX() *pprofX {
	p := &pprofX{
		name: "go tool pprof",
		arg:  &ArgPprofX{},
	}
	p.arg.setName(p.name)
	return p
}

type ArgPprofX struct {
	commonArg
}

func (pa ArgPprofX) build() (s []string) {
	s = []string{
		pa.name,
		"-no_browser=true",
		fmt.Sprintf(HTTPTpl, pa.port),
		pa.file,
	}
	return
}
func (pa ArgPprofX) AppendEnv(s []string) []string {
	return s
}
