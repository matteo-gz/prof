package biz

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DealRun1 run run
func (uc *Usecase) DealRun1(ctx context.Context, uri string) (relativePath string, err error) {
	uri, err = url.QueryUnescape(uri)
	if err != nil {
		return
	}
	relativePath, err = uc.curlOne(ctx, uri)
	return
}
func (uc *Usecase) curlOne(ctx context.Context, uri string) (relativePath string, err error) {
	data, err := curlGet(ctx, uri, 60*5)
	if err != nil {
		return
	}
	return uc.repo.CreateFile(uri, data)
}

func curlGet(ctx context.Context, uri string, timeout int) (data []byte, err error) {
	client := &http.Client{Timeout: time.Second * time.Duration(timeout)}
	req, err := http.NewRequestWithContext(ctx, "GET", uri, http.NoBody)
	if err != nil {
		return
	}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	data, err = io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		err = errors.New(uri + "\n" + string(data))
	}
	return
}
