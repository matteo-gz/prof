package tool

import (
	"fmt"
	"strings"
)

type traceX struct {
	name string
	arg  *ArgTraceX
}

func newTraceX() *traceX {
	t := &traceX{
		name: traceName(),
		arg:  &ArgTraceX{},
	}
	t.arg.setName(t.name)
	return t
}

func (t traceX) Command() string {
	return strings.Join(t.arg.build(), Sep)
}
func (t traceX) Arg() arg {
	return t.arg
}

type ArgTraceX struct {
	commonArg
}

func (at ArgTraceX) AppendEnv(s []string) []string {
	return traceEnv(s)
}
func (at ArgTraceX) build() (s []string) {
	s = []string{
		at.name,
		fmt.Sprintf(HTTPTpl, at.port),
		at.file,
	}
	return
}
