package biz

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/matteo-gz/prof/pkg/pproftype"
	"net/url"
	"strconv"
	"sync"
)

type batch struct {
	oriUrl   string
	fileList []string
	seconds  int
	url      string
	urlList  []string
	log      *log.Helper
}

const maxTime = 180

func newBatch(url string, log *log.Helper) *batch {
	return &batch{
		oriUrl: url,
		log:    log,
	}
}

type saveFile func(url string, contentType string, data []byte) (relativePath string, err error)

func (b *batch) Start(ctx context.Context, fn saveFile) (res []cse, err error) {
	if err = b.setUrl(); err != nil {
		return
	}
	b.setUrlList()
	wg := sync.WaitGroup{}
	taskL := len(b.urlList)
	wg.Add(taskL)
	ch := make(chan cse, taskL)
	for _, v := range b.urlList {
		go b.run(ctx, v, ch, &wg, fn)
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

// ces combine string error
type cse struct {
	S string
	E error
}

func (b *batch) run(ctx context.Context, url string, ch chan cse, wg *sync.WaitGroup, fn saveFile) {
	defer func() {
		wg.Done()

	}()
	data, contentType, err := curlGet(ctx, url, b.seconds+5)
	if err != nil {
		ch <- cse{"", err}
		return
	}
	relativePath, err := fn(url, contentType, data)
	if err != nil {
		ch <- cse{"", err}
		return
	}
	ch <- cse{relativePath, nil}
	return
}

func (uc *Usecase) DealRun(ctx context.Context, uri string) (res []cse, err error) {
	b := newBatch(uri, uc.log)
	return b.Start(ctx, uc.repo.CreateFile)
}
func (b *batch) setUrl() (err error) {
	uri, err := url.QueryUnescape(b.oriUrl)
	if err != nil {
		return
	}
	urlP, err := url.Parse(uri)
	if err != nil {
		return
	}
	q := urlP.Query()
	seconds := q.Get("seconds")
	q.Del("seconds")
	urlP.RawQuery = q.Encode()
	b.url = urlP.String()
	if seconds == "" {
		seconds = "1"
	}
	b.seconds, err = strconv.Atoi(seconds)
	if err == nil && maxTime < b.seconds {
		b.seconds = maxTime
	}
	return
}
func (b *batch) setUrlList() {
	for _, v := range pproftype.List {
		u := fmt.Sprintf("%s/%s?seconds=%d", b.url, v, b.seconds)
		b.urlList = append(b.urlList, u)
	}
}
