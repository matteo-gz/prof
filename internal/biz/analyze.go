package biz

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/google/pprof/profile"
)

type SampleTypeInfo struct {
	Type string `json:"type"`
	Unit string `json:"unit"`
}

type TopItem struct {
	Rank    int     `json:"rank"`
	Name    string  `json:"name"`
	Flat    int64   `json:"flat"`
	FlatPct float64 `json:"flat_pct"`
	SumPct  float64 `json:"sum_pct"`
	Cum     int64   `json:"cum"`
	CumPct  float64 `json:"cum_pct"`
}

type TopResult struct {
	Path        string           `json:"path"`
	Type        string           `json:"type"`
	SampleType  SampleTypeInfo   `json:"sample_type"`
	SampleTypes []SampleTypeInfo `json:"sample_types"`
	Total       int64            `json:"total"`
	Unit        string           `json:"unit"`
	Top         []TopItem        `json:"top"`
}

type funcStat struct {
	FuncID   uint64
	Name     string
	Flat     int64
	Cum      int64
}

func AnalyzeTop(absPath string, n int, sampleIdx int, cumSort bool) (*TopResult, error) {
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

	stats, total := aggregateFlatCum(p, idx)

	if cumSort {
		sort.Slice(stats, func(i, j int) bool {
			if stats[i].Cum != stats[j].Cum {
				return stats[i].Cum > stats[j].Cum
			}
			return stats[i].Name < stats[j].Name
		})
	} else {
		sort.Slice(stats, func(i, j int) bool {
			if stats[i].Flat != stats[j].Flat {
				return stats[i].Flat > stats[j].Flat
			}
			return stats[i].Name < stats[j].Name
		})
	}

	if n <= 0 {
		n = 20
	}
	if n > len(stats) {
		n = len(stats)
	}

	items := make([]TopItem, n)
	var sumPct float64
	for i := 0; i < n; i++ {
		s := stats[i]
		flatPct := pct(s.Flat, total)
		sumPct += flatPct
		items[i] = TopItem{
			Rank:    i + 1,
			Name:    s.Name,
			Flat:    s.Flat,
			FlatPct: roundPct(flatPct),
			SumPct:  roundPct(sumPct),
			Cum:     s.Cum,
			CumPct:  roundPct(pct(s.Cum, total)),
		}
	}

	sampleTypes := make([]SampleTypeInfo, len(p.SampleType))
	for i, st := range p.SampleType {
		sampleTypes[i] = SampleTypeInfo{Type: st.Type, Unit: st.Unit}
	}

	return &TopResult{
		Path:        absPath,
		Type:        inferType(absPath),
		SampleType:  SampleTypeInfo{Type: p.SampleType[idx].Type, Unit: p.SampleType[idx].Unit},
		SampleTypes: sampleTypes,
		Total:       total,
		Unit:        p.SampleType[idx].Unit,
		Top:         items,
	}, nil
}

func resolveSampleIndex(p *profile.Profile, si int) int {
	if si >= 0 && si < len(p.SampleType) {
		return si
	}
	return len(p.SampleType) - 1
}

func aggregateFlatCum(p *profile.Profile, idx int) ([]funcStat, int64) {
	flatMap := make(map[uint64]int64)
	cumMap := make(map[uint64]int64)
	nameMap := make(map[uint64]string)
	var total int64

	for _, s := range p.Sample {
		if idx >= len(s.Value) {
			continue
		}
		v := s.Value[idx]
		total += v

		// flat: leaf function only (Location[0], innermost Line)
		if len(s.Location) > 0 {
			if fn := leafFunction(s.Location[0]); fn != nil {
				flatMap[fn.ID] += v
				nameMap[fn.ID] = fn.Name
			}
		}

		// cum: every unique function on the stack
		seen := make(map[uint64]bool)
		for _, loc := range s.Location {
			for _, line := range loc.Line {
				if line.Function == nil {
					continue
				}
				fid := line.Function.ID
				if seen[fid] {
					continue
				}
				seen[fid] = true
				cumMap[fid] += v
				nameMap[fid] = line.Function.Name
			}
		}
	}

	allIDs := make(map[uint64]bool)
	for id := range flatMap {
		allIDs[id] = true
	}
	for id := range cumMap {
		allIDs[id] = true
	}

	stats := make([]funcStat, 0, len(allIDs))
	for id := range allIDs {
		flat, cum := flatMap[id], cumMap[id]
		if flat == 0 && cum == 0 {
			continue
		}
		stats = append(stats, funcStat{
			FuncID: id,
			Name:   nameMap[id],
			Flat:   flat,
			Cum:    cum,
		})
	}
	return stats, total
}

func leafFunction(loc *profile.Location) *profile.Function {
	if len(loc.Line) == 0 {
		return nil
	}
	return loc.Line[0].Function
}

func pct(value, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(value) / float64(total) * 100
}

func roundPct(v float64) float64 {
	return math.Round(v*100) / 100
}

func inferType(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "cpu"):
		return "cpu"
	case strings.Contains(lower, "heap"):
		return "heap"
	case strings.Contains(lower, "alloc"):
		return "allocs"
	case strings.Contains(lower, "goroutine"):
		return "goroutine"
	case strings.Contains(lower, "block"):
		return "block"
	case strings.Contains(lower, "mutex"):
		return "mutex"
	case strings.Contains(lower, "threadcreate"):
		return "threadcreate"
	default:
		return "unknown"
	}
}
