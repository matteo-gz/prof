package biz

import (
	"strings"
	"testing"

	"github.com/matteo-gz/prof/pkg/pproftype"
)

func taskURLTypes(tasks []urlTask) []string {
	var types []string
	for _, t := range tasks {
		for _, pt := range pproftype.List {
			if strings.Contains(t.url, "/"+pt+"?") || strings.HasSuffix(t.url, "/"+pt) {
				types = append(types, pt)
				break
			}
		}
	}
	return types
}

func hasType(types []string, name string) bool {
	for _, t := range types {
		if t == name {
			return true
		}
	}
	return false
}

func TestBatchTypes_EmptySelectsAll(t *testing.T) {
	b := newBatchFromParams(BatchParams{
		URL:             "http://localhost:6060/debug/pprof",
		SamplingSeconds: 10,
		SnapshotMode:    "snapshot",
		DeltaSeconds:    10,
		TraceEnabled:    true,
		TraceSeconds:    5,
		Types:           nil,
	}, nil, false, 30, 10, 5)
	b.url = b.oriUrl
	b.setUrlList()

	types := taskURLTypes(b.taskList)
	for _, pt := range pproftype.List {
		if !hasType(types, pt) {
			t.Errorf("expected type %s in task list when Types is empty", pt)
		}
	}
}

func TestBatchTypes_GoroutineHeap(t *testing.T) {
	b := newBatchFromParams(BatchParams{
		URL:             "http://localhost:6060/debug/pprof",
		SamplingSeconds: 10,
		SnapshotMode:    "snapshot",
		DeltaSeconds:    10,
		TraceSeconds:    5,
		Types:           []string{"goroutine", "heap"},
	}, nil, false, 30, 10, 5)
	b.url = b.oriUrl
	b.setUrlList()

	types := taskURLTypes(b.taskList)
	expect := map[string]bool{
		pproftype.Goroutine: true,
		pproftype.Heap:      true,
		pproftype.Cmdline:   true,
		pproftype.Symbol:    true,
	}
	if len(types) != len(expect) {
		t.Fatalf("expected %d tasks, got %d: %v", len(expect), len(types), types)
	}
	for _, tp := range types {
		if !expect[tp] {
			t.Errorf("unexpected type %s in task list", tp)
		}
	}
}

func TestBatchTypes_ProfileOnly(t *testing.T) {
	b := newBatchFromParams(BatchParams{
		URL:             "http://localhost:6060/debug/pprof",
		SamplingSeconds: 10,
		SnapshotMode:    "snapshot",
		DeltaSeconds:    10,
		TraceSeconds:    5,
		Types:           []string{"profile"},
	}, nil, false, 30, 10, 5)
	b.url = b.oriUrl
	b.setUrlList()

	types := taskURLTypes(b.taskList)
	expect := map[string]bool{
		pproftype.Profile: true,
		pproftype.Cmdline: true,
		pproftype.Symbol:  true,
	}
	if len(types) != len(expect) {
		t.Fatalf("expected %d tasks, got %d: %v", len(expect), len(types), types)
	}
	for _, tp := range types {
		if !expect[tp] {
			t.Errorf("unexpected type %s in task list", tp)
		}
	}
}

func TestBatchTypes_TraceAutoEnabled(t *testing.T) {
	b := newBatchFromParams(BatchParams{
		URL:             "http://localhost:6060/debug/pprof",
		SamplingSeconds: 10,
		SnapshotMode:    "snapshot",
		DeltaSeconds:    10,
		TraceEnabled:    false,
		TraceSeconds:    5,
		Types:           []string{"trace"},
	}, nil, false, 30, 10, 5)
	b.url = b.oriUrl
	b.setUrlList()

	if !b.traceEnabled {
		t.Error("expected traceEnabled=true when Types contains trace")
	}
	types := taskURLTypes(b.taskList)
	expect := map[string]bool{
		pproftype.Trace:   true,
		pproftype.Cmdline: true,
		pproftype.Symbol:  true,
	}
	if len(types) != len(expect) {
		t.Fatalf("expected %d tasks, got %d: %v", len(expect), len(types), types)
	}
	for _, tp := range types {
		if !expect[tp] {
			t.Errorf("unexpected type %s in task list", tp)
		}
	}
	for _, task := range b.taskList {
		if strings.Contains(task.url, "/trace?") {
			if !strings.Contains(task.url, "seconds=5") {
				t.Errorf("trace URL should contain seconds param, got: %s", task.url)
			}
		}
	}
}

func TestBatchTypes_InvalidTypesIgnored(t *testing.T) {
	b := newBatchFromParams(BatchParams{
		URL:             "http://localhost:6060/debug/pprof",
		SamplingSeconds: 10,
		SnapshotMode:    "snapshot",
		DeltaSeconds:    10,
		TraceSeconds:    5,
		Types:           []string{"invalid_type", "not_real"},
	}, nil, false, 30, 10, 5)
	b.url = b.oriUrl
	b.setUrlList()

	if b.selectedTypes != nil {
		t.Error("expected selectedTypes to be nil when all types are invalid")
	}
	types := taskURLTypes(b.taskList)
	if len(types) < len(pproftype.List)-1 {
		t.Errorf("expected all types when invalid types given, got %d: %v", len(types), types)
	}
}
