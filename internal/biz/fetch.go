package biz

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const curlTimeout = 300 // 5 minutes

func (uc *Usecase) DealRun1(ctx context.Context, uri string) (relativePath string, err error) {
	uri, err = url.QueryUnescape(uri)
	if err != nil {
		return
	}
	relativePath, err = uc.curlOne(ctx, uri)
	return
}
func validateURL(uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return errors.New("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("only http/https allowed")
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()) {
		return errors.New("private/loopback IP not allowed")
	}
	return nil
}

func (uc *Usecase) curlOne(ctx context.Context, uri string) (relativePath string, err error) {
	if err = validateURL(uri); err != nil {
		return
	}
	data, contentType, err := curlGet(ctx, uri, curlTimeout)
	if err != nil {
		return
	}
	return uc.repo.CreateFile(uri, contentType, data)
}

func curlGet(ctx context.Context, uri string, timeout int) (data []byte, contentType string, err error) {
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
	contentType = res.Header.Get("Content-Type")
	return
}
