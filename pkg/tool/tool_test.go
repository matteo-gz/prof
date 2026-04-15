package tool

import (
	"testing"

	"github.com/matteo-gz/prof/pkg/pproftype"
)

func TestNewTool_Pprof(t *testing.T) {
	toolx := NewTool(pproftype.ExtPprof)
	toolx.Arg().SetPort(11)
	toolx.Arg().SetFile("a.log")
	got := toolx.Command()
	want := "go tool pprof -no_browser=true -http=0.0.0.0:11 a.log"
	if got != want {
		t.Errorf("pprof command = %q, want %q", got, want)
	}
}

func TestNewTool_Trace(t *testing.T) {
	tool2 := NewTool(pproftype.ExtTrace)
	tool2.Arg().SetPort(11)
	tool2.Arg().SetFile("a.log")
	got := tool2.Command()
	want := "BROWSER=echo go tool trace -http=0.0.0.0:11 a.log"
	if got != want {
		t.Errorf("trace command = %q, want %q", got, want)
	}
}
