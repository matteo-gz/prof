package biz

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/pprof/profile"
)

// FlameStack is one sampled stack (same semantics as google/pprof/internal/report.Stack).
type FlameStack struct {
	Value   int64 `json:"value"`
	Sources []int `json:"sources"`
}

// FlamePlace references one slot in FlameResult.Stacks (aligned with pprof StackSlot).
type FlamePlace struct {
	Stack int `json:"stack"`
	Pos   int `json:"pos"`
}

// FlameSource is one frame identity in the flame graph (aligned with pprof StackSource).
type FlameSource struct {
	FullName   string       `json:"full_name"`
	FileName   string       `json:"file_name"`
	UniqueName string       `json:"unique_name"`
	Inlined    bool         `json:"inlined"`
	Display    []string     `json:"display"`
	Places     []FlamePlace `json:"places"`
	Self       int64        `json:"self"`
	Color      int          `json:"color"`
}

// FlameResult is the JSON payload for the flame graph view (pprof Web /flamegraph data model).
type FlameResult struct {
	Path        string           `json:"path"`
	Type        string           `json:"type"`
	SampleType  SampleTypeInfo   `json:"sample_type"`
	SampleTypes []SampleTypeInfo `json:"sample_types"`
	Total       int64            `json:"total"`
	Unit        string           `json:"unit"`
	Scale       float64          `json:"scale"`
	Stacks      []FlameStack     `json:"stacks"`
	Sources     []FlameSource    `json:"sources"`
}

var flameSepRE = regexp.MustCompile(`::|\.`)
var flameFileSepRE = regexp.MustCompile(`/`)

// AnalyzeFlame builds stack list + source table like pprof's report.StackSet (see google-pprof/internal/report/stacks.go).
func AnalyzeFlame(absPath string, sampleIdx int, trimPathOpt, sourcePath string) (*FlameResult, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("open profile: %w", err)
	}
	defer f.Close()

	p, err := profile.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}

	if len(p.SampleType) == 0 {
		return nil, fmt.Errorf("profile has no sample types")
	}

	idx := resolveSampleIndex(p, sampleIdx)
	if idx < 0 || idx >= len(p.SampleType) {
		return nil, fmt.Errorf("sample index %d out of range [0, %d)", sampleIdx, len(p.SampleType))
	}

	sampleTypes := make([]SampleTypeInfo, len(p.SampleType))
	for i, st := range p.SampleType {
		sampleTypes[i] = SampleTypeInfo{Type: st.Type, Unit: st.Unit}
	}

	type srcKey struct {
		funcName string
		fileName string
		line     int64
		column   int64
		inlined  bool
	}
	srcIndex := map[srcKey]int{}
	seenFunctions := map[string]bool{}
	unknownIndex := 1

	out := &FlameResult{
		Path:        absPath,
		Type:        inferType(absPath),
		SampleType:  SampleTypeInfo{Type: p.SampleType[idx].Type, Unit: p.SampleType[idx].Unit},
		SampleTypes: sampleTypes,
		Unit:        p.SampleType[idx].Unit,
		Scale:       1,
		Stacks:      []FlameStack{},
		Sources: []FlameSource{{
			FullName: "root",
			Display:  []string{"root"},
			Places:   []FlamePlace{},
		}},
	}

	getSrc := func(line profile.Line, inlined bool) int {
		fn := line.Function
		if fn == nil {
			fn = &profile.Function{Name: fmt.Sprintf("?%d?", unknownIndex)}
			unknownIndex++
		}

		k := srcKey{fn.Name, fn.Filename, line.Line, line.Column, inlined}
		if i, ok := srcIndex[k]; ok {
			return i
		}

		fileName := trimPathForFlame(fn.Filename, trimPathOpt, sourcePath)
		fs := FlameSource{
			FileName: fileName,
			Inlined:  inlined,
			Places:   []FlamePlace{},
		}
		if fn.Name != "" {
			fs.FullName = addLineInfoFlame(fn.Name, line)
			fs.Display = shortNameListFlame(fs.FullName)
			fs.Color = pickColorFlame(packageNameFlame(fn.Name))
		} else {
			fs.FullName = addLineInfoFlame(fileName, line)
			fs.Display = fileNameSuffixesFlame(fs.FullName)
			fs.Color = pickColorFlame(filepath.Dir(fileName))
		}

		if !seenFunctions[fs.FullName] {
			fs.UniqueName = fs.FullName
			seenFunctions[fs.FullName] = true
		} else {
			fs.UniqueName = fmt.Sprint(fs.FullName, "#", fn.ID)
		}

		out.Sources = append(out.Sources, fs)
		i := len(out.Sources) - 1
		srcIndex[k] = i
		return i
	}

	var total int64
	for _, sample := range p.Sample {
		if idx >= len(sample.Value) {
			continue
		}
		v := sample.Value[idx]
		total += v

		st := FlameStack{Value: v, Sources: []int{0}}
		for i := len(sample.Location) - 1; i >= 0; i-- {
			loc := sample.Location[i]
			for j := len(loc.Line) - 1; j >= 0; j-- {
				line := loc.Line[j]
				inlined := j != len(loc.Line)-1
				st.Sources = append(st.Sources, getSrc(line, inlined))
			}
		}

		leaf := st.Sources[len(st.Sources)-1]
		out.Sources[leaf].Self += v
		out.Stacks = append(out.Stacks, st)
	}

	fillPlacesFlame(out)

	out.Total = total
	return out, nil
}

func fillPlacesFlame(s *FlameResult) {
	for si, stack := range s.Stacks {
		seen := map[int]bool{}
		for pos, src := range stack.Sources {
			if seen[src] {
				continue
			}
			seen[src] = true
			s.Sources[src].Places = append(s.Sources[src].Places, FlamePlace{Stack: si, Pos: pos})
		}
	}
}

func addLineInfoFlame(str string, line profile.Line) string {
	if line.Column != 0 {
		return fmt.Sprint(str, ":", line.Line, ":", line.Column)
	}
	if line.Line != 0 {
		return fmt.Sprint(str, ":", line.Line)
	}
	return str
}

func allSuffixesFlame(name string, re *regexp.Regexp) []string {
	seps := re.FindAllStringIndex(name, -1)
	result := make([]string, 0, len(seps)+1)
	result = append(result, name)
	for _, sep := range seps {
		if sep[1] < len(name) {
			result = append(result, name[sep[1]:])
		}
	}
	return result
}

func shortNameListFlame(name string) []string {
	return allSuffixesFlame(name, flameSepRE)
}

func fileNameSuffixesFlame(name string) []string {
	if name == "" {
		return []string{""}
	}
	return allSuffixesFlame(filepath.ToSlash(filepath.Clean(name)), flameFileSepRE)
}

var flamePkgRE = regexp.MustCompile(`^((.*/)?[\w\d_]+)(\.|::)([^/]*)$`)

func packageNameFlame(name string) string {
	m := flamePkgRE.FindStringSubmatch(name)
	if m == nil {
		return ""
	}
	return m[1]
}

func pickColorFlame(key string) int {
	const numColors = 1048576
	h := sha256.Sum256([]byte(key))
	index := binary.LittleEndian.Uint32(h[:])
	return int(index % numColors)
}

// trimPathForFlame mirrors google/pprof/internal/report.trimPath (simplified import path).
func trimPathForFlame(path, trimPath, searchPath string) string {
	sPath := filepath.ToSlash(path)
	if trimPath == "" {
		for _, dir := range filepath.SplitList(searchPath) {
			want := "/" + filepath.Base(dir) + "/"
			if found := strings.Index(sPath, want); found != -1 {
				return path[found+len(want):]
			}
		}
	}
	trimPaths := append(filepath.SplitList(filepath.ToSlash(trimPath)), "/proc/self/cwd/./", "/proc/self/cwd/")
	for _, tp := range trimPaths {
		if !strings.HasSuffix(tp, "/") {
			tp += "/"
		}
		if strings.HasPrefix(sPath, tp) {
			return path[len(tp):]
		}
	}
	return path
}
