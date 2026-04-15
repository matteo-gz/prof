package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/matteo-gz/prof/internal/biz"
	"github.com/matteo-gz/prof/pkg/pproftype"
	"github.com/matteo-gz/prof/web"
)

const (
	RoutePprofPre = "/pprof"
	RouteTracePre = "/trace"
	RouteTrace    = RouteTracePre + "/%s/"
	RoutePprof    = RoutePprofPre + "/%s/"
)

func (s *Service) Index(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	var buf bytes.Buffer
	data := map[string]string{
		"Port2": s.port2,
	}
	if err := s.tmpl.ExecuteTemplate(&buf, "index.html", data); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.String(http.StatusOK, buf.String())
}

func (s *Service) Css(c *gin.Context) {
	data, err := web.StaticFS.ReadFile("static/bootstrap.min.css")
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Data(http.StatusOK, "text/css; charset=utf-8", data)
}

func (s *Service) PersonCurl(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, "person_curl.html", nil); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.String(http.StatusOK, buf.String())
}

func (s *Service) FileList(c *gin.Context) {
	dir := c.Query("dir")
	lists, files, _ := s.uc.GetFileList(dir)
	data := gin.H{
		"list":  lists,
		"dir":   dir,
		"files": files,
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, "list.html", data); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.String(http.StatusOK, buf.String())
}

func (s *Service) File(c *gin.Context) {
	dir := c.Query("dir")
	dir = filepath.Clean(dir)
	if strings.Contains(dir, "..") {
		c.String(http.StatusBadRequest, "invalid path")
		return
	}
	absPath := s.uc.GetAbsDir(dir)
	absPath = filepath.Clean(absPath)
	storageRoot := filepath.Clean(s.uc.GetAbsDir("/"))
	if !strings.HasPrefix(absPath, storageRoot) {
		c.String(http.StatusBadRequest, "path out of range")
		return
	}

	ext := s.uc.GetFileType(dir)
	if ext == pproftype.ExtTxt {
		data, err := os.ReadFile(absPath)
		if err != nil {
			c.String(http.StatusOK, err.Error())
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, "%s", string(data))
		return
	} else if ext == pproftype.ExtUnknown {
		c.String(http.StatusOK, "unknown file type")
	} else {
		enDir := base64.StdEncoding.EncodeToString([]byte(dir))
		var route string
		if ext == pproftype.ExtTrace {
			route = RouteTrace
		} else {
			route = RoutePprof
		}
		urls := fmt.Sprintf(route, enDir)
		c.Redirect(http.StatusTemporaryRedirect, urls)
		return
	}
}

func newUri(c *gin.Context) biz.Uri {
	return biz.Uri{
		Path:  c.Request.URL.Path,
		Dir:   c.Param("dir"),
		Query: c.Request.URL.Query().Encode(),
	}
}

func (s *Service) PprofProxy(c *gin.Context) {
	u := newUri(c)
	u.Route = RoutePprof
	u.ProxyBasePath = "/ui/"
	u.ReBody = func(body string, currPath string) string {
		body = strings.ReplaceAll(body, "href=\"./", "href=\""+currPath)
		return body
	}
	s.log.Debugf("%#v", u)
	err := s.uc.Proxy(u, c.Writer, c.Request)
	if err != nil {
		c.String(404, err.Error())
	}
}

func (s *Service) TraceProxy(c *gin.Context) {
	u := newUri(c)
	u.Route = RouteTrace
	u.ProxyBasePath = "/"
	u.ReBody = func(body string, currPath string) string {
		body = strings.ReplaceAll(body, "href=\"/", "href=\""+currPath)
		body = strings.ReplaceAll(body, "src=\"/", "src=\""+currPath)
		body = strings.ReplaceAll(body, "action=\"/", "action=\""+currPath)
		body = strings.ReplaceAll(body, "getJSON('/", "getJSON('"+currPath)
		body = strings.ReplaceAll(body, "url = '/", "url = '"+currPath)
		return body
	}
	s.log.Debugf("%#v", u)
	err := s.uc.Proxy(u, c.Writer, c.Request)
	if err != nil {
		c.String(404, err.Error())
	}
}

func (s *Service) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	relativePath, err := s.uc.DealUpload(file, header.Filename)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"res": []string{relativePath},
		"id":  time.Now().UnixMilli(),
	})
}

func (s *Service) Run(c *gin.Context) {
	uri := c.PostForm("url")
	res, err := s.uc.DealRun(c.Request.Context(), uri)
	if err != nil {
		c.JSON(200, gin.H{
			"res": []string{err.Error()},
		})
		return
	}
	var res2 []string
	for i := range res {
		if res[i].E != nil {
			res2 = append(res2, res[i].E.Error())
		} else {
			res2 = append(res2, res[i].S)
		}
	}
	c.JSON(200, gin.H{
		"id":  time.Now().UnixMilli(),
		"res": res2,
	})
}

func (s *Service) Run1(c *gin.Context) {
	uri := c.PostForm("url")
	relativePath, err := s.uc.DealRun1(c.Request.Context(), uri)
	var res string
	if err != nil {
		res = err.Error()
	} else {
		res = relativePath
	}
	c.JSON(200, gin.H{
		"id":  time.Now().UnixMilli(),
		"res": []string{res},
	})
}
