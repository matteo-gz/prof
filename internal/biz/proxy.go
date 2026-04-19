package biz

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

type pp struct {
	Route         string
	ProxyBasePath string
	ReBody        replaceBody
}
type cp struct {
	Route         string
	ProxyBasePath string
	ReBody        replaceBody
}
type px struct {
}

type replaceBody func(body []byte, currPath string) []byte

var proxyBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}
type Uri struct {
	Path          string
	Dir           string
	Base          string
	Query         string
	Route         string
	ProxyBasePath string
	ReBody        replaceBody
}

func (uc *Usecase) Proxy(u Uri, rw http.ResponseWriter, req *http.Request) (err error) {
	usePort, err := uc.getPortByDir(u.Dir)
	if err != nil {
		return
	}
	currPath := fmt.Sprintf(u.Route, u.Dir)
	path := strings.Replace(u.Path, currPath, "", 1)
	urlX := fmt.Sprintf("http://127.0.0.1:%d%s%s?%s", usePort, u.ProxyBasePath, path, u.Query)
	uc.log.Debugf("call:%s", urlX)
	urlT, err := url.Parse(urlX)
	if err != nil {
		return
	}
	director := func(req *http.Request) {
		req.URL = urlT
		req.Host = urlT.Host
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ModifyResponse = func(res *http.Response) error {
		if res.StatusCode != 200 {
			return nil
		}
		buf := proxyBufPool.Get().(*bytes.Buffer)
		buf.Reset()
		if _, err := buf.ReadFrom(res.Body); err != nil {
			proxyBufPool.Put(buf)
			return err
		}
		transformed := u.ReBody(buf.Bytes(), currPath)
		result := make([]byte, len(transformed))
		copy(result, transformed)
		proxyBufPool.Put(buf)
		res.Body = io.NopCloser(bytes.NewReader(result))
		res.ContentLength = int64(len(result))
		res.Header.Set("Content-Length", strconv.Itoa(len(result)))
		return nil
	}
	proxy.ServeHTTP(rw, req)
	return nil
}

func (uc *Usecase) ProxyDiff(u Uri, rw http.ResponseWriter, req *http.Request) (err error) {
	baseBytes, err := base64.StdEncoding.DecodeString(u.Base)
	if err != nil {
		return
	}
	compareBytes, err := base64.StdEncoding.DecodeString(u.Dir)
	if err != nil {
		return
	}
	usePort, err := uc.repo.GetPortByDiff(string(baseBytes), string(compareBytes))
	if err != nil {
		return
	}
	currPath := fmt.Sprintf(u.Route, u.Base, u.Dir)
	path := strings.Replace(u.Path, currPath, "", 1)
	urlX := fmt.Sprintf("http://127.0.0.1:%d%s%s?%s", usePort, u.ProxyBasePath, path, u.Query)
	uc.log.Debugf("call:%s", urlX)
	urlT, err := url.Parse(urlX)
	if err != nil {
		return
	}
	director := func(req *http.Request) {
		req.URL = urlT
		req.Host = urlT.Host
	}
	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ModifyResponse = func(res *http.Response) error {
		if res.StatusCode != 200 {
			return nil
		}
		buf := proxyBufPool.Get().(*bytes.Buffer)
		buf.Reset()
		if _, err := buf.ReadFrom(res.Body); err != nil {
			proxyBufPool.Put(buf)
			return err
		}
		transformed := u.ReBody(buf.Bytes(), currPath)
		result := make([]byte, len(transformed))
		copy(result, transformed)
		proxyBufPool.Put(buf)
		res.Body = io.NopCloser(bytes.NewReader(result))
		res.ContentLength = int64(len(result))
		res.Header.Set("Content-Length", strconv.Itoa(len(result)))
		return nil
	}
	proxy.ServeHTTP(rw, req)
	return nil
}
