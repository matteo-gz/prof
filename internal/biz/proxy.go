package biz

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
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

type replaceBody func(body string, currPath string) string
type Uri struct {
	Path          string
	Dir           string
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
		if res.StatusCode == 200 {
			oldPayload, err := io.ReadAll(res.Body)
			if err != nil {
				return err
			}
			newPayLoadStr := string(oldPayload)
			newPayLoadStr = u.ReBody(newPayLoadStr, currPath)
			newPayLoad := []byte(newPayLoadStr)
			res.Body = io.NopCloser(bytes.NewBuffer(newPayLoad))
			res.ContentLength = int64(len(newPayLoad))
			res.Header.Set("Content-Length", fmt.Sprint(len(newPayLoad)))
			return nil
		} else {
			return nil
		}
	}
	proxy.ServeHTTP(rw, req)
	return nil
}
