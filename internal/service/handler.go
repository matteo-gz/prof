package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/matteo-gz/prof/internal/biz"
	"github.com/matteo-gz/prof/pkg/pproftype"
	"github.com/matteo-gz/prof/web"
)

const (
	RoutePprofPre     = "/pprof"
	RouteTracePre     = "/trace"
	RoutePprofDiffPre = "/pprof-diff"
	RouteTrace        = RouteTracePre + "/%s/"
	RoutePprof        = RoutePprofPre + "/%s/"
	RoutePprofDiff    = RoutePprofDiffPre + "/%s/%s/"
)

func (s *Service) Index(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	var buf bytes.Buffer
	data := map[string]any{
		"Port2":           s.port2,
		"SamplingSeconds": s.samplingSeconds,
		"DeltaSeconds":    s.deltaSeconds,
		"TraceSeconds":    s.traceSeconds,
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

func (s *Service) StaticFile(c *gin.Context) {
	fp := c.Param("filepath")
	data, err := web.StaticFS.ReadFile("static" + fp)
	if err != nil {
		c.String(http.StatusNotFound, "not found")
		return
	}
	if strings.HasSuffix(fp, ".js") {
		c.Data(http.StatusOK, "application/javascript; charset=utf-8", data)
	} else if strings.HasSuffix(fp, ".css") {
		c.Data(http.StatusOK, "text/css; charset=utf-8", data)
	} else {
		c.Data(http.StatusOK, "application/octet-stream", data)
	}
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

func isDateDir(dir string) bool {
	base := dir
	if idx := strings.LastIndex(dir, "/"); idx >= 0 {
		base = dir[idx+1:]
	}
	if len(base) != 8 {
		return false
	}
	for _, c := range base {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (s *Service) FileList(c *gin.Context) {
	dir := c.Query("dir")
	lists, files, _ := s.uc.GetFileList(dir)
	data := gin.H{
		"list":  lists,
		"dir":   dir,
		"files": files,
	}
	if isDateDir(dir) {
		if groups, err := s.uc.GetBatchGroups(dir); err == nil && len(groups) > 0 {
			data["groups"] = groups
			data["list"] = nil
		}
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

const pluginLoaderTag = `<script src="/static/plugins/loader.js"></script>`

func injectPlugins(body string) string {
	if idx := strings.Index(body, "</head>"); idx >= 0 {
		return body[:idx] + pluginLoaderTag + body[idx:]
	}
	return body
}

var pluginLoaderTagBytes = []byte(pluginLoaderTag)
var headCloseTag = []byte("</head>")

func injectPluginsBytes(body []byte) []byte {
	idx := bytes.Index(body, headCloseTag)
	if idx < 0 {
		return body
	}
	out := make([]byte, 0, len(body)+len(pluginLoaderTagBytes))
	out = append(out, body[:idx]...)
	out = append(out, pluginLoaderTagBytes...)
	out = append(out, body[idx:]...)
	return out
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
	u.ReBody = func(body []byte, currPath string) []byte {
		body = bytes.ReplaceAll(body, []byte(`href="./`), []byte(`href="`+currPath))
		body = injectPluginsBytes(body)
		return body
	}
	s.log.Debugf("%#v", u)
	err := s.uc.Proxy(u, c.Writer, c.Request)
	if err != nil {
		c.String(404, err.Error())
	}
}

func (s *Service) PprofDiffProxy(c *gin.Context) {
	u := newUri(c)
	u.Base = c.Param("base")
	u.Route = RoutePprofDiff
	u.ProxyBasePath = "/ui/"
	u.ReBody = func(body []byte, currPath string) []byte {
		body = bytes.ReplaceAll(body, []byte(`href="./`), []byte(`href="`+currPath))
		body = injectPluginsBytes(body)
		return body
	}
	s.log.Debugf("%#v", u)
	err := s.uc.ProxyDiff(u, c.Writer, c.Request)
	if err != nil {
		c.String(404, err.Error())
	}
}

func (s *Service) TraceProxy(c *gin.Context) {
	u := newUri(c)
	u.Route = RouteTrace
	u.ProxyBasePath = "/"
	u.ReBody = func(body []byte, currPath string) []byte {
		cp := []byte(currPath)
		body = bytes.ReplaceAll(body, []byte(`href="/`), append([]byte(`href="`), cp...))
		body = bytes.ReplaceAll(body, []byte(`src="/`), append([]byte(`src="`), cp...))
		body = bytes.ReplaceAll(body, []byte(`action="/`), append([]byte(`action="`), cp...))
		body = bytes.ReplaceAll(body, []byte("getJSON('/"), append([]byte("getJSON('"), cp...))
		body = bytes.ReplaceAll(body, []byte("url = '/"), append([]byte("url = '"), cp...))
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
	ctx := c.Request.Context()
	ct := c.ContentType()

	var res []biz.Cse
	var err error

	if strings.Contains(ct, "application/json") {
		var p biz.BatchParams
		if err = c.ShouldBindJSON(&p); err != nil {
			c.JSON(200, gin.H{"res": []string{"invalid json: " + err.Error()}})
			return
		}
		res, err = s.uc.DealRun(ctx, p)
	} else {
		uri := c.PostForm("url")
		res, err = s.uc.DealRunLegacy(ctx, uri)
	}

	if err != nil {
		c.JSON(200, gin.H{"res": []string{err.Error()}})
		return
	}
	var res2 []string
	var files []biz.FileResult
	for i := range res {
		if res[i].E != nil {
			res2 = append(res2, res[i].E.Error())
		} else {
			res2 = append(res2, res[i].S)
			files = append(files, biz.FileResult{Path: res[i].S, Size: res[i].Size})
		}
	}
	c.JSON(200, gin.H{
		"id":    time.Now().UnixMilli(),
		"res":   res2,
		"files": files,
	})
}

type PluginInfo struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

func (s *Service) APIPlugins(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"plugins": []PluginInfo{
			{
				Name:        "i18n-zh",
				Enabled:     s.pluginsConfig.I18nZh,
				Description: "界面中文汉化",
			},
			{
				Name:        "source-fold",
				Enabled:     s.pluginsConfig.SourceFold,
				Description: "Source 视图函数折叠",
			},
		{
			Name:        "peek-fold",
			Enabled:     s.pluginsConfig.PeekFold,
			Description: "Peek 视图分组折叠",
		},
		{
			Name:        "graph-explain",
			Enabled:     s.pluginsConfig.GraphExplain,
			Description: "Graph 节点点击解释",
		},
	},
	})
}

type apiEndpoint struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Desc   string            `json:"description"`
	Params []apiParam        `json:"params,omitempty"`
}

type apiParam struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
	Default  string `json:"default,omitempty"`
	Desc     string `json:"description"`
}

func (s *Service) APIIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "Prof API",
		"version": "1.0",
		"docs":    "https://github.com/matteo-gz/prof",
		"endpoints": []apiEndpoint{
			{
				Method: "GET",
				Path:   "/api/",
				Desc:   "API 索引，返回所有可用接口列表",
			},
			{
				Method: "GET",
				Path:   "/api/plugins",
				Desc:   "获取已注册的 JS 插件列表及启用状态",
			},
			{
				Method: "GET",
				Path:   "/api/pprof/:dir/top",
				Desc:   "解析 .pprof 文件，返回 Top N 热点函数（flat/cum）",
				Params: []apiParam{
					{Name: "dir", In: "path", Required: true, Desc: "profile 相对路径的 base64 编码（与 /pprof/:dir/ 相同）"},
					{Name: "n", In: "query", Default: "20", Desc: "返回前 N 个函数"},
					{Name: "si", In: "query", Default: "-1", Desc: "SampleType 索引，-1 = 最后一个（与 go tool pprof 一致）"},
					{Name: "cum", In: "query", Default: "false", Desc: "true 时按 cum 排序，否则按 flat 排序"},
				},
			},
		},
	})
}

func (s *Service) APIPprofTop(c *gin.Context) {
	dir := c.Param("dir")
	n, _ := strconv.Atoi(c.DefaultQuery("n", "20"))
	si, _ := strconv.Atoi(c.DefaultQuery("si", "-1"))
	cumSort := c.Query("cum") == "true"

	result, err := s.uc.AnalyzeTop(dir, n, si, cumSort)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Service) Run1(c *gin.Context) {
	uri := c.PostForm("url")
	relativePath, size, err := s.uc.DealRun1(c.Request.Context(), uri)
	if err != nil {
		c.JSON(200, gin.H{
			"id":  time.Now().UnixMilli(),
			"res": []string{err.Error()},
		})
		return
	}
	c.JSON(200, gin.H{
		"id":    time.Now().UnixMilli(),
		"res":   []string{relativePath},
		"files": []biz.FileResult{{Path: relativePath, Size: size}},
	})
}
