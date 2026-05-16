package biz

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/google/pprof/profile"
)

// PeekEdge is one caller or callee edge (aligned with pprof peek: weight / pct / name + file:line + optional inline).
type PeekEdge struct {
	Name     string  `json:"name"`
	File     string  `json:"file,omitempty"`
	Line     int64   `json:"line,omitempty"`
	Inline   bool    `json:"inline"`
	Weight   int64   `json:"weight"`
	PctOfCum float64 `json:"pct_of_cum"`
}

type PeekMatch struct {
	Name    string     `json:"name"`
	File    string     `json:"file,omitempty"`
	Line    int64      `json:"line,omitempty"`
	Flat          int64      `json:"flat"`
	Cum           int64      `json:"cum"`
	FlatPct       float64    `json:"flat_pct"`
	CumPct        float64    `json:"cum_pct"`
	Callers       []PeekEdge `json:"callers"`
	Callees       []PeekEdge `json:"callees"`
}

type PeekResult struct {
	Path        string           `json:"path"`
	Type        string           `json:"type"`
	SampleType  SampleTypeInfo   `json:"sample_type"`
	SampleTypes []SampleTypeInfo `json:"sample_types"`
	Total       int64            `json:"total"`
	Unit        string           `json:"unit"`
	Filter      string           `json:"filter"`
	Matches     []PeekMatch      `json:"matches"`
}

// nodeKey identifies one graph node at line granularity (same idea as graph.NodeInfo for lines mode).
type nodeKey struct {
	FuncID uint64
	File   string
	Line   int64
}

type edgeKE struct {
	From, To nodeKey
}

type frameNode struct {
	Key    nodeKey
	Name   string
	LocID  uint64
	Ni     int // index in Location.Line
	NLines int // len(Line)
}

func AnalyzePeek(absPath, filterPattern string, sampleIdx int) (*PeekResult, error) {
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

	flat := make(map[nodeKey]int64)
	cum := make(map[nodeKey]int64)
	meta := make(map[nodeKey]string) // short function name
	edges := make(map[edgeKE]struct {
		W      int64
		Inline bool
	})
	var total int64

	for _, s := range p.Sample {
		if idx >= len(s.Value) {
			continue
		}
		v := s.Value[idx]
		total += v

		chain := buildFrameChain(s)
		if len(chain) == 0 {
			continue
		}

		seenN := make(map[nodeKey]bool)
		for _, fr := range chain {
			if seenN[fr.Key] {
				continue
			}
			seenN[fr.Key] = true
			cum[fr.Key] += v
			meta[fr.Key] = fr.Name
		}

		leaf := chain[len(chain)-1].Key
		flat[leaf] += v

		seenE := make(map[edgeKE]bool)
		for j := 0; j < len(chain)-1; j++ {
			a, b := chain[j], chain[j+1]
			ek := edgeKE{From: a.Key, To: b.Key}
			if seenE[ek] {
				continue
			}
			seenE[ek] = true
			// Mirror graph.newGraph AddToEdgeDiv(..., ni != len(locNodes)-1): callee is b.
			inline := b.NLines > 1 && b.Ni != b.NLines-1
			e := edges[ek]
			e.W += v
			e.Inline = e.Inline || inline
			edges[ek] = e
		}
	}

	allK := make(map[nodeKey]bool)
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
			delete(meta, k)
		}
	}

	var keys []nodeKey
	for k := range meta {
		pn := nodePrintableName(k, meta[k])
		if !re.MatchString(pn) && !re.MatchString(meta[k]) {
			continue
		}
		if cum[k] == 0 && flat[k] == 0 {
			continue
		}
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a := nodePrintableName(keys[i], meta[keys[i]])
		b := nodePrintableName(keys[j], meta[keys[j]])
		return a < b
	})

	if len(keys) == 0 {
		return nil, fmt.Errorf("no functions match filter: %s", filterPattern)
	}

	sampleTypes := make([]SampleTypeInfo, len(p.SampleType))
	for i, st := range p.SampleType {
		sampleTypes[i] = SampleTypeInfo{Type: st.Type, Unit: st.Unit}
	}

	out := &PeekResult{
		Path:        absPath,
		Type:        inferType(absPath),
		SampleType:  SampleTypeInfo{Type: p.SampleType[idx].Type, Unit: p.SampleType[idx].Unit},
		SampleTypes: sampleTypes,
		Total:       total,
		Unit:        p.SampleType[idx].Unit,
		Filter:      filterPattern,
		Matches:     nil,
	}

	for _, nk := range keys {
		nodeCum := cum[nk]
		var callers []PeekEdge
		for ek, ew := range edges {
			if ek.To != nk || ew.W == 0 {
				continue
			}
			callers = append(callers, peekEdgeFrom(ek.From, meta, ew.W, nodeCum, ew.Inline))
		}
		sort.Slice(callers, func(i, j int) bool {
			if callers[i].Weight != callers[j].Weight {
				return callers[i].Weight > callers[j].Weight
			}
			return peekEdgeLess(callers[i], callers[j])
		})

		var callees []PeekEdge
		for ek, ew := range edges {
			if ek.From != nk || ew.W == 0 {
				continue
			}
			callees = append(callees, peekEdgeFrom(ek.To, meta, ew.W, nodeCum, ew.Inline))
		}
		sort.Slice(callees, func(i, j int) bool {
			if callees[i].Weight != callees[j].Weight {
				return callees[i].Weight > callees[j].Weight
			}
			return peekEdgeLess(callees[i], callees[j])
		})

		fl, cm := flat[nk], cum[nk]
		nm := meta[nk]
		out.Matches = append(out.Matches, PeekMatch{
			Name:          nm,
			File:          nk.File,
			Line:          nk.Line,
			Flat:          fl,
			Cum:           cm,
			FlatPct:       roundPct(pct(fl, total)),
			CumPct:        roundPct(pct(cm, total)),
			Callers:       callers,
			Callees:       callees,
		})
	}

	return out, nil
}

func peekEdgeFrom(k nodeKey, meta map[nodeKey]string, w, nodeCum int64, inline bool) PeekEdge {
	nm := meta[k]
	return PeekEdge{
		Name:     nm,
		File:     k.File,
		Line:     k.Line,
		Inline:   inline,
		Weight:   w,
		PctOfCum: roundPct(pct(w, nodeCum)),
	}
}

func peekEdgeLess(a, b PeekEdge) bool {
	if a.Name != b.Name {
		return a.Name < b.Name
	}
	if a.File != b.File {
		return a.File < b.File
	}
	return a.Line < b.Line
}

// nodePrintableName follows google/pprof/internal/graph.NodeInfo.NameComponents for line granularity (no PC hex).
func nodePrintableName(k nodeKey, funcName string) string {
	var parts []string
	if funcName != "" {
		parts = append(parts, funcName)
	}
	switch {
	case k.Line != 0 && k.File != "":
		s := fmt.Sprintf("%s:%d", k.File, k.Line)
		parts = append(parts, s)
	case k.File != "":
		parts = append(parts, k.File)
	case funcName != "":
	default:
		parts = append(parts, "<unknown>")
	}
	return strings.Join(parts, " ")
}

// buildFrameChain mirrors google/pprof/internal/graph.newGraph stack expansion order.
func buildFrameChain(s *profile.Sample) []frameNode {
	var chain []frameNode
	for i := len(s.Location) - 1; i >= 0; i-- {
		loc := s.Location[i]
		lines := loc.Line
		if len(lines) == 0 {
			continue
		}
		nLines := len(lines)
		for ni := len(lines) - 1; ni >= 0; ni-- {
			ln := lines[ni]
			if ln.Function == nil {
				continue
			}
			fn := ln.Function
			key := nodeKey{FuncID: fn.ID, File: fn.Filename, Line: ln.Line}
			chain = append(chain, frameNode{
				Key:    key,
				Name:   fn.Name,
				LocID:  loc.ID,
				Ni:     ni,
				NLines: nLines,
			})
		}
	}
	return chain
}
