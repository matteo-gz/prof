package biz

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/matteo-gz/prof/pkg/pproftype"
)

// BatchParams holds the structured parameters for a batch profiling run.
type BatchParams struct {
	URL             string   `json:"url"`
	SamplingSeconds int      `json:"sampling_seconds"`
	SnapshotMode    string   `json:"snapshot_mode"`
	DeltaSeconds    int      `json:"delta_seconds"`
	TraceEnabled    bool     `json:"trace_enabled"`
	TraceSeconds    int      `json:"trace_seconds"`
	Types           []string `json:"types"`
}

type urlTask struct {
	url     string
	timeout int
}

type batch struct {
	oriUrl          string
	url             string
	samplingSeconds int
	snapshotMode    string
	deltaSeconds    int
	traceEnabled    bool
	traceSeconds    int
	selectedTypes   map[string]bool
	fileList        []string
	taskList        []urlTask
	log             *log.Helper
	denyPrivateIP   bool
}

const maxTime = 180

func clampSeconds(v, defaultVal int) int {
	if v <= 0 {
		return defaultVal
	}
	if v > maxTime {
		return maxTime
	}
	return v
}

func buildSelectedTypes(types []string) map[string]bool {
	if len(types) == 0 {
		return nil
	}
	m := make(map[string]bool, len(types))
	for _, t := range types {
		if pproftype.IsValidType(t) {
			m[t] = true
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

func newBatchFromParams(p BatchParams, log *log.Helper, denyPrivateIP bool, defaultSampling, defaultDelta, defaultTrace int) *batch {
	mode := p.SnapshotMode
	if mode != "delta" {
		mode = "snapshot"
	}
	sel := buildSelectedTypes(p.Types)
	traceEnabled := p.TraceEnabled
	if sel != nil {
		traceEnabled = sel[pproftype.Trace]
	}
	return &batch{
		oriUrl:          p.URL,
		samplingSeconds: clampSeconds(p.SamplingSeconds, defaultSampling),
		snapshotMode:    mode,
		deltaSeconds:    clampSeconds(p.DeltaSeconds, defaultDelta),
		traceEnabled:    traceEnabled,
		traceSeconds:    clampSeconds(p.TraceSeconds, defaultTrace),
		selectedTypes:   sel,
		log:             log,
		denyPrivateIP:   denyPrivateIP,
	}
}

// newBatchLegacy creates a batch from a legacy URL containing ?seconds=N.
func newBatchLegacy(rawURL string, log *log.Helper, denyPrivateIP bool, defaultSampling, defaultDelta, defaultTrace int) *batch {
	return &batch{
		oriUrl:          rawURL,
		samplingSeconds: defaultSampling,
		deltaSeconds:    defaultDelta,
		traceEnabled:    false,
		traceSeconds:    defaultTrace,
		snapshotMode:    "snapshot",
		log:             log,
		denyPrivateIP:   denyPrivateIP,
	}
}

type saveFile func(url string, contentType string, data []byte) (relativePath string, err error)

func (b *batch) Start(ctx context.Context, fn saveFile) (res []Cse, err error) {
	if err = b.setUrl(); err != nil {
		return
	}
	b.setUrlList()
	wg := sync.WaitGroup{}
	taskL := len(b.taskList)
	wg.Add(taskL)
	ch := make(chan Cse, taskL)
	for _, t := range b.taskList {
		go b.run(ctx, t, ch, &wg, fn)
	}
	wg.Wait()
	for {
		if taskL == 0 {
			break
		}
		taskL--
		res = append(res, <-ch)
	}
	return
}

// Cse combines a string result, file size, and an error.
type Cse struct {
	S    string
	Size int64
	E    error
}

// FileResult holds path and size for a successfully fetched file.
type FileResult struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func (b *batch) run(ctx context.Context, task urlTask, ch chan Cse, wg *sync.WaitGroup, fn saveFile) {
	defer wg.Done()
	data, contentType, err := curlGet(ctx, task.url, task.timeout)
	if err != nil {
		ch <- Cse{"", 0, err}
		return
	}
	size := int64(len(data))
	relativePath, err := fn(task.url, contentType, data)
	if err != nil {
		ch <- Cse{"", 0, err}
		return
	}
	ch <- Cse{relativePath, size, nil}
}

func (uc *Usecase) DealRun(ctx context.Context, p BatchParams) (res []Cse, err error) {
	b := newBatchFromParams(p, uc.log, uc.denyPrivateIP, uc.samplingSeconds, uc.deltaSeconds, uc.traceSeconds)
	return b.Start(ctx, uc.repo.CreateFile)
}

// DealRunLegacy handles the old URL-based batch request for backward compatibility.
func (uc *Usecase) DealRunLegacy(ctx context.Context, uri string) (res []Cse, err error) {
	b := newBatchLegacy(uri, uc.log, uc.denyPrivateIP, uc.samplingSeconds, uc.deltaSeconds, uc.traceSeconds)
	return b.Start(ctx, uc.repo.CreateFile)
}

func (b *batch) setUrl() (err error) {
	uri, err := url.QueryUnescape(b.oriUrl)
	if err != nil {
		return
	}
	if err = validateURL(uri, b.denyPrivateIP); err != nil {
		return
	}
	urlP, err := url.Parse(uri)
	if err != nil {
		return
	}
	q := urlP.Query()

	// Legacy path: extract seconds from URL if present and sampling/delta not yet set.
	if seconds := q.Get("seconds"); seconds != "" {
		q.Del("seconds")
		if v, e := strconv.Atoi(seconds); e == nil {
			if b.samplingSeconds <= 0 {
				b.samplingSeconds = clampSeconds(v, 30)
			}
			if b.deltaSeconds <= 0 {
				b.deltaSeconds = clampSeconds(v, 10)
			}
		}
	}

	urlP.RawQuery = q.Encode()
	b.url = urlP.String()
	return
}

func (b *batch) setUrlList() {
	for _, v := range pproftype.List {
		family := pproftype.Family(v)

		if b.selectedTypes != nil && family != pproftype.FamilyMeta {
			if !b.selectedTypes[v] {
				continue
			}
		}

		var u string
		var timeout int
		switch family {
		case pproftype.FamilySampling:
			u = fmt.Sprintf("%s/%s?seconds=%d", b.url, v, b.samplingSeconds)
			timeout = b.samplingSeconds + 5
		case pproftype.FamilyTrace:
			if !b.traceEnabled {
				continue
			}
			u = fmt.Sprintf("%s/%s?seconds=%d", b.url, v, b.traceSeconds)
			timeout = b.traceSeconds + 5
		case pproftype.FamilySnapshot:
			if b.snapshotMode == "delta" {
				u = fmt.Sprintf("%s/%s?seconds=%d", b.url, v, b.deltaSeconds)
				timeout = b.deltaSeconds + 5
			} else {
				u = fmt.Sprintf("%s/%s", b.url, v)
				timeout = 10
			}
		case pproftype.FamilyMeta:
			u = fmt.Sprintf("%s/%s", b.url, v)
			timeout = 5
		}
		b.taskList = append(b.taskList, urlTask{url: u, timeout: timeout})
	}
}
