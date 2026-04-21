package biz

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// FlameLayoutCell is one rectangle in the merged flame tree: row depth, column among siblings (weight-desc).
type FlameLayoutCell struct {
	Name     string  `json:"name"`
	Row      int     `json:"row"`       // 1-based: root row = 1
	Col      int     `json:"col"`       // 1-based among siblings under the same parent (sorted by value desc)
	Pct      float64 `json:"pct"`       // share of profile total (0–100)
	Value    int64   `json:"value"`     // raw aggregated value through this node
	Size     string  `json:"size"`      // human-readable amount (e.g. "2.73")
	SizeUnit string  `json:"size_unit"` // e.g. "MB", "KB", "ns"
	SrcIndex int     `json:"src_index"` // index into raw /flame sources
	FullName string  `json:"full_name,omitempty"`
}

// FlameLayoutResult merges per-sample stacks into one tree and assigns row/col like the Web flame view
// (root top; under each parent, wider/heavier callees to the left → col 1,2,3…).
type FlameLayoutResult struct {
	Path        string              `json:"path"`
	Type        string              `json:"type"`
	SampleType  SampleTypeInfo      `json:"sample_type"`
	SampleTypes []SampleTypeInfo    `json:"sample_types"`
	Total       int64               `json:"total"`
	Unit        string              `json:"unit"`
	Cells       []FlameLayoutCell   `json:"cells"`
	Rows        [][]FlameLayoutCell `json:"rows"` // rows[i] = row (i+1); each slice sorted by col
}

type flameTrieNode struct {
	srcIdx int
	val    int64
	kids   map[int]*flameTrieNode
}

func newFlameTrie() *flameTrieNode {
	return &flameTrieNode{srcIdx: 0, kids: make(map[int]*flameTrieNode)}
}

func mergeStackIntoTrie(root *flameTrieNode, sources []int, v int64) {
	if len(sources) == 0 || v == 0 {
		return
	}
	cur := root
	cur.val += v
	for i := 1; i < len(sources); i++ {
		sid := sources[i]
		next, ok := cur.kids[sid]
		if !ok {
			next = &flameTrieNode{srcIdx: sid, kids: make(map[int]*flameTrieNode)}
			cur.kids[sid] = next
		}
		next.val += v
		cur = next
	}
}

func sortedTrieChildren(n *flameTrieNode) []*flameTrieNode {
	if len(n.kids) == 0 {
		return nil
	}
	list := make([]*flameTrieNode, 0, len(n.kids))
	for _, ch := range n.kids {
		list = append(list, ch)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].val != list[j].val {
			return list[i].val > list[j].val
		}
		return list[i].srcIdx < list[j].srcIdx
	})
	return list
}

func layoutTrieDFS(n *flameTrieNode, depth, col int, raw *FlameResult, total int64, out *[]FlameLayoutCell) {
	row := depth + 1
	name, full := flameDisplayNames(raw, n.srcIdx)
	sz, su := formatFlameSize(n.val, raw.Unit)
	p := pctFlame(n.val, total)
	*out = append(*out, FlameLayoutCell{
		Name: name, Row: row, Col: col, Pct: p, Value: n.val,
		Size: sz, SizeUnit: su, SrcIndex: n.srcIdx, FullName: full,
	})
	for i, ch := range sortedTrieChildren(n) {
		layoutTrieDFS(ch, depth+1, i+1, raw, total, out)
	}
}

func flameDisplayNames(raw *FlameResult, srcIdx int) (shortName, fullName string) {
	if raw == nil || srcIdx < 0 || srcIdx >= len(raw.Sources) {
		return "?", ""
	}
	s := raw.Sources[srcIdx]
	fullName = s.FullName
	if len(s.Display) > 0 {
		shortName = s.Display[len(s.Display)-1]
	} else {
		shortName = s.FullName
	}
	if shortName == "" {
		shortName = "?"
	}
	return shortName, fullName
}

func pctFlame(v, total int64) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(v)/float64(total)*10000) / 100
}

func formatFlameSize(v int64, unit string) (amount string, sizeUnit string) {
	u := strings.ToLower(strings.TrimSpace(unit))
	switch u {
	case "bytes":
		const kb int64 = 1024
		const mb = kb * 1024
		const gb = mb * 1024
		switch {
		case v >= gb:
			return fmt.Sprintf("%.2f", float64(v)/float64(gb)), "GB"
		case v >= mb:
			return fmt.Sprintf("%.2f", float64(v)/float64(mb)), "MB"
		case v >= kb:
			return fmt.Sprintf("%.2f", float64(v)/float64(kb)), "KB"
		default:
			return strconv.FormatInt(v, 10), "B"
		}
	case "nanoseconds":
		switch {
		case v >= 1e9:
			return fmt.Sprintf("%.2f", float64(v)/1e9), "s"
		case v >= 1e6:
			return fmt.Sprintf("%.2f", float64(v)/1e6), "ms"
		case v >= 1e3:
			return fmt.Sprintf("%.2f", float64(v)/1e3), "µs"
		default:
			return strconv.FormatInt(v, 10), "ns"
		}
	default:
		return strconv.FormatInt(v, 10), unit
	}
}

func groupCellsByRow(cells []FlameLayoutCell) [][]FlameLayoutCell {
	maxRow := 0
	for _, c := range cells {
		if c.Row > maxRow {
			maxRow = c.Row
		}
	}
	if maxRow == 0 {
		return nil
	}
	rows := make([][]FlameLayoutCell, maxRow)
	for _, c := range cells {
		i := c.Row - 1
		rows[i] = append(rows[i], c)
	}
	for i := range rows {
		sort.Slice(rows[i], func(a, b int) bool {
			if rows[i][a].Col != rows[i][b].Col {
				return rows[i][a].Col < rows[i][b].Col
			}
			return rows[i][a].Name < rows[i][b].Name
		})
	}
	return rows
}

// AnalyzeFlameLayout parses the profile once, merges stacks into a trie, emits cells with row/col + formatted size.
func AnalyzeFlameLayout(absPath string, sampleIdx int, trimPathOpt, sourcePath string) (*FlameLayoutResult, error) {
	raw, err := AnalyzeFlame(absPath, sampleIdx, trimPathOpt, sourcePath)
	if err != nil {
		return nil, err
	}

	root := newFlameTrie()
	for _, st := range raw.Stacks {
		mergeStackIntoTrie(root, st.Sources, st.Value)
	}

	var cells []FlameLayoutCell
	layoutTrieDFS(root, 0, 1, raw, raw.Total, &cells)

	sort.Slice(cells, func(i, j int) bool {
		if cells[i].Row != cells[j].Row {
			return cells[i].Row < cells[j].Row
		}
		if cells[i].Col != cells[j].Col {
			return cells[i].Col < cells[j].Col
		}
		return cells[i].Name < cells[j].Name
	})

	out := &FlameLayoutResult{
		Path:        raw.Path,
		Type:        raw.Type,
		SampleType:  raw.SampleType,
		SampleTypes: raw.SampleTypes,
		Total:       raw.Total,
		Unit:        raw.Unit,
		Cells:       cells,
		Rows:        groupCellsByRow(cells),
	}
	return out, nil
}
