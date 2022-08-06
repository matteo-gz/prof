package tool

import (
	"fmt"
	"github.com/matteo-gz/prof/pkg/pproftype"
	"testing"
)

func TestNewTool(t *testing.T) {
	toolx := NewTool(pproftype.ExtPprof)
	toolx.Arg().SetPort(11)
	toolx.Arg().SetFile("a.log")
	s := toolx.Command()
	if s == "go tool pprof -no_browser=true -http=0.0.0.0:11 a.log" {
		fmt.Println("ok")
	}

	tool2 := NewTool(pproftype.ExtTrace)
	tool2.Arg().SetPort(11)
	tool2.Arg().SetFile("a.log")
	s2 := tool2.Command()
	if s2 == "go tool trace -http=0.0.0.0:11 a.log" {
		fmt.Println("ok")
	}
}
