package biz

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/google/pprof/profile"
)

// lineStatKey identifies one profiling node at source line granularity (similar to graph.Node).
type lineStatKey struct {
	File string
	Line int64
	Name string
}

type SourceLine struct {
	Line    int64   `json:"line"`
	Text    string  `json:"text"`
	Flat    int64   `json:"flat"`
	Cum     int64   `json:"cum"`
	FlatPct float64 `json:"flat_pct"`
	CumPct  float64 `json:"cum_pct"`
}

type SourceFuncBlock struct {
	Name       string       `json:"name"`
	FlatSum    int64        `json:"flat_sum"`
	CumSum     int64        `json:"cum_sum"`
	Lines      []SourceLine `json:"lines"`
}

type SourceFileBlock struct {
	Path      string            `json:"path"`
	Error     string            `json:"error,omitempty"`
	Functions []SourceFuncBlock `json:"functions"`
}

type SourceResult struct {
	Path        string            `json:"path"`
	Type        string            `json:"type"`
	SampleType  SampleTypeInfo    `json:"sample_type"`
	SampleTypes []SampleTypeInfo  `json:"sample_types"`
	Total       int64             `json:"total"`
	Unit        string            `json:"unit"`
	Filter      string            `json:"filter"`
	Truncated   bool              `json:"truncated,omitempty"`
	MaxFiles    int               `json:"max_files,omitempty"`
	Files       []SourceFileBlock `json:"files"`
}

// DefaultSourceMaxFiles aligns with google/pprof internal/driver maxEntries for weblist (50).
const DefaultSourceMaxFiles = 50

func AnalyzeSource(absPath, filterPattern string, sampleIdx, margin int, extraSearchPath string, maxFiles int) (*SourceResult, error) {
	re, err := regexp.Compile(filterPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid filter regexp: %w", err)
	}

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

	if margin <= 0 {
		margin = 5
	}

	searchDirs := sourceSearchDirs(extraSearchPath)

	flatMap, cumMap, total := aggregateLineSamples(p, idx)

	matched := make(map[lineStatKey]bool)
	for k := range flatMap {
		if re.MatchString(k.Name) {
			matched[k] = true
		}
	}
	for k := range cumMap {
		if re.MatchString(k.Name) {
			matched[k] = true
		}
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("no functions match filter: %s", filterPattern)
	}

	byFileThenName := make(map[string]map[string][]lineStatKey)
	for k := range matched {
		fm := k.File
		if fm == "" {
			continue
		}
		if byFileThenName[fm] == nil {
			byFileThenName[fm] = make(map[string][]lineStatKey)
		}
		byFileThenName[fm][k.Name] = append(byFileThenName[fm][k.Name], k)
	}

	if len(byFileThenName) == 0 {
		return nil, fmt.Errorf("matched samples have no source file paths in profile")
	}

	if maxFiles <= 0 {
		maxFiles = DefaultSourceMaxFiles
	}

	type fileScore struct {
		path  string
		score int64
	}
	scores := make([]fileScore, 0, len(byFileThenName))
	for fp, funcs := range byFileThenName {
		var s int64
		for _, keys := range funcs {
			for _, k := range keys {
				s += flatMap[k] + cumMap[k]
			}
		}
		scores = append(scores, fileScore{path: fp, score: s})
	}
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score != scores[j].score {
			return scores[i].score > scores[j].score
		}
		return scores[i].path < scores[j].path
	})
	truncated := len(scores) > maxFiles
	if truncated {
		scores = scores[:maxFiles]
	}

	filePaths := make([]string, len(scores))
	for i := range scores {
		filePaths[i] = scores[i].path
	}

	sampleTypes := make([]SampleTypeInfo, len(p.SampleType))
	for i, st := range p.SampleType {
		sampleTypes[i] = SampleTypeInfo{Type: st.Type, Unit: st.Unit}
	}

	out := SourceResult{
		Path:        absPath,
		Type:        inferType(absPath),
		SampleType:  SampleTypeInfo{Type: p.SampleType[idx].Type, Unit: p.SampleType[idx].Unit},
		SampleTypes: sampleTypes,
		Total:       total,
		Unit:        p.SampleType[idx].Unit,
		Filter:      filterPattern,
		Truncated:   truncated,
		MaxFiles:    maxFiles,
		Files:       nil,
	}

	for _, logicalPath := range filePaths {
		funcs := byFileThenName[logicalPath]
		names := make([]string, 0, len(funcs))
		for n := range funcs {
			names = append(names, n)
		}
		sort.Strings(names)

		absFile, openErr := openSourceFile(logicalPath, searchDirs)
		fileBlock := SourceFileBlock{Path: logicalPath, Functions: nil}
		if openErr != nil {
			fileBlock.Error = openErr.Error()
			out.Files = append(out.Files, fileBlock)
			continue
		}

		allLines, readErr := readAllLines(absFile)
		_ = absFile.Close()
		if readErr != nil {
			fileBlock.Error = readErr.Error()
			out.Files = append(out.Files, fileBlock)
			continue
		}

		for _, fnName := range names {
			keys := funcs[fnName]
			var minL, maxL int64
			first := true
			for _, k := range keys {
				line := k.Line
				if line <= 0 {
					continue
				}
				if first {
					minL, maxL = line, line
					first = false
				} else {
					if line < minL {
						minL = line
					}
					if line > maxL {
						maxL = line
					}
				}
			}
			if first {
				continue
			}
			start := minL - int64(margin)
			if start < 1 {
				start = 1
			}
			end := maxL + int64(margin)

			var flatSum, cumSum int64
			var linesOut []SourceLine
			for ln := start; ln <= end; ln++ {
				k := lineStatKey{File: logicalPath, Line: ln, Name: fnName}
				fl := flatMap[k]
				cm := cumMap[k]
				flatSum += fl
				cumSum += cm
				txt := ""
				if int(ln) < len(allLines) {
					txt = allLines[ln]
				}
				linesOut = append(linesOut, SourceLine{
					Line:    ln,
					Text:    txt,
					Flat:    fl,
					Cum:     cm,
					FlatPct: roundPct(pct(fl, total)),
					CumPct:  roundPct(pct(cm, total)),
				})
			}
			if len(linesOut) == 0 {
				continue
			}

			fileBlock.Functions = append(fileBlock.Functions, SourceFuncBlock{
				Name:    fnName,
				FlatSum: flatSum,
				CumSum:  cumSum,
				Lines:   linesOut,
			})
		}

		if fileBlock.Error != "" || len(fileBlock.Functions) > 0 {
			out.Files = append(out.Files, fileBlock)
		}
	}

	return &out, nil
}

func aggregateLineSamples(p *profile.Profile, idx int) (flat, cum map[lineStatKey]int64, total int64) {
	flat = make(map[lineStatKey]int64)
	cum = make(map[lineStatKey]int64)

	for _, s := range p.Sample {
		if idx >= len(s.Value) {
			continue
		}
		v := s.Value[idx]
		total += v

		if len(s.Location) > 0 && len(s.Location[0].Line) > 0 {
			ln := s.Location[0].Line[0]
			if ln.Function != nil {
				k := lineStatKey{File: ln.Function.Filename, Line: ln.Line, Name: ln.Function.Name}
				flat[k] += v
			}
		}

		seen := make(map[lineStatKey]bool)
		for _, loc := range s.Location {
			for _, ln := range loc.Line {
				if ln.Function == nil {
					continue
				}
				k := lineStatKey{File: ln.Function.Filename, Line: ln.Line, Name: ln.Function.Name}
				if seen[k] {
					continue
				}
				seen[k] = true
				cum[k] += v
			}
		}
	}

	allK := make(map[lineStatKey]bool)
	for k := range flat {
		allK[k] = true
	}
	for k := range cum {
		allK[k] = true
	}
	for k := range allK {
		if flat[k] == 0 && cum[k] == 0 {
			delete(flat, k)
			delete(cum, k)
		}
	}
	return flat, cum, total
}

func sourceSearchDirs(extra string) []string {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	var dirs []string
	if extra != "" {
		for _, p := range filepath.SplitList(extra) {
			p = strings.TrimSpace(p)
			if p != "" {
				dirs = append(dirs, p)
			}
		}
	}
	dirs = append(dirs, wd)
	return dirs
}

func openSourceFile(logicalPath string, searchDirs []string) (*os.File, error) {
	if logicalPath == "" {
		return nil, fmt.Errorf("empty source path in profile")
	}
	if filepath.IsAbs(logicalPath) {
		return os.Open(logicalPath)
	}
	for _, dir := range searchDirs {
		full := filepath.Join(dir, logicalPath)
		if f, err := os.Open(full); err == nil {
			return f, nil
		}
	}
	return nil, fmt.Errorf("source file not found (try source_path=): %s", logicalPath)
}

func readAllLines(f *os.File) ([]string, error) {
	var lines []string
	lines = append(lines, "")
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, sc.Err()
}
