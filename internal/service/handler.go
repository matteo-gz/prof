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
		"ProfBin":         s.profBin,
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
	Method          string         `json:"method"`
	Path            string         `json:"path"`
	Desc            string         `json:"description"`
	Params          []apiParam     `json:"params,omitempty"`
	ResponseSummary string         `json:"response_summary,omitempty"`
	ResponseSchema  []apiField     `json:"response_schema,omitempty"`
	ResponseExample map[string]any `json:"response_example,omitempty"`
}

type apiParam struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
	Default  string `json:"default,omitempty"`
	Desc     string `json:"description"`
}

type apiField struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Desc string `json:"description"`
}

func (s *Service) APIIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "Prof API",
		"version": "1.0",
		"docs":    "https://github.com/matteo-gz/prof",
		"endpoints": []apiEndpoint{
			{
				Method:          "GET",
				Path:            "/api/",
				Desc:            "API 索引，返回所有可用接口列表",
				ResponseSummary: "返回 API 元信息与 endpoint 列表",
				ResponseSchema: []apiField{
					{Name: "name", Type: "string", Desc: "API 名称"},
					{Name: "version", Type: "string", Desc: "API 版本"},
					{Name: "docs", Type: "string", Desc: "项目文档地址"},
					{Name: "endpoints", Type: "array<object>", Desc: "接口列表（每项含 method/path/description/params/response_*）"},
				},
				ResponseExample: map[string]any{
					"name":    "Prof API",
					"version": "1.0",
					"docs":    "https://github.com/matteo-gz/prof",
					"endpoints": []map[string]any{
						{"method": "GET", "path": "/api/plugins"},
					},
				},
			},
			{
				Method:          "GET",
				Path:            "/api/plugins",
				Desc:            "获取已注册的 JS 插件列表及启用状态",
				ResponseSummary: "返回插件名、开关状态与用途说明",
				ResponseSchema: []apiField{
					{Name: "plugins", Type: "array<object>", Desc: "插件数组"},
					{Name: "plugins[].name", Type: "string", Desc: "插件名"},
					{Name: "plugins[].enabled", Type: "bool", Desc: "是否启用"},
					{Name: "plugins[].description", Type: "string", Desc: "插件说明"},
				},
				ResponseExample: map[string]any{
					"plugins": []map[string]any{
						{"name": "i18n-zh", "enabled": true, "description": "界面中文汉化"},
					},
				},
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
				ResponseSummary: "返回当前采样类型下的 top 函数列表与占比",
				ResponseSchema: []apiField{
					{Name: "path", Type: "string", Desc: "profile 相对路径（base64 解码后）"},
					{Name: "type", Type: "string", Desc: "profile 类型推断值"},
					{Name: "sample_type", Type: "object", Desc: "当前采样类型（type/unit）"},
					{Name: "sample_types", Type: "array<object>", Desc: "全部采样类型"},
					{Name: "total", Type: "int64", Desc: "当前采样类型总值"},
					{Name: "unit", Type: "string", Desc: "单位（如 bytes/count/nanoseconds）"},
					{Name: "top", Type: "array<object>", Desc: "热点函数列表"},
					{Name: "top[].rank/name/flat/cum/flat_pct/sum_pct/cum_pct", Type: "mixed", Desc: "排名、函数名、flat/cum 及百分比"},
				},
				ResponseExample: map[string]any{
					"path":        "/20260420/23_34_52/heap",
					"sample_type": map[string]any{"type": "inuse_space", "unit": "bytes"},
					"total":       6043872,
					"unit":        "bytes",
					"top": []map[string]any{
						{"rank": 1, "name": "runtime.mallocgc", "flat": 2100281, "flat_pct": 34.75, "sum_pct": 34.75, "cum": 2100281, "cum_pct": 34.75},
					},
				},
			},
			{
				Method: "GET",
				Path:   "/api/pprof/:dir/source",
				Desc:   "解析 .pprof，按源码行聚合 flat/cum；需 f 正则过滤函数名",
				Params: []apiParam{
					{Name: "dir", In: "path", Required: true, Desc: "profile 相对路径 base64"},
					{Name: "f", In: "query", Default: "(空)", Desc: "函数名正则；空字符串与 go tool pprof Web 一致，等价匹配全部符号"},
					{Name: "si", In: "query", Default: "-1", Desc: "SampleType 索引"},
					{Name: "margin", In: "query", Default: "5", Desc: "采样行前后附加展示行数"},
					{Name: "max_files", In: "query", Default: "50", Desc: "最多返回多少个源文件（按采样权重排序），与 pprof weblist 上限一致"},
					{Name: "source_path", In: "query", Default: "", Desc: "源码搜索路径（可多路径，与 PATH 分隔符一致），找不到文件时必填"},
				},
				ResponseSummary: "返回按源码文件/函数/行聚合后的 flat/cum 与源码片段",
				ResponseSchema: []apiField{
					{Name: "path/type/sample_type/sample_types/total/unit/filter", Type: "mixed", Desc: "通用元信息与过滤条件"},
					{Name: "files", Type: "array<object>", Desc: "源文件列表，按权重排序"},
					{Name: "files[].path", Type: "string", Desc: "源码文件路径"},
					{Name: "files[].functions", Type: "array<object>", Desc: "文件内函数聚合"},
					{Name: "files[].functions[].name/flat/cum", Type: "mixed", Desc: "函数及统计值"},
					{Name: "files[].functions[].lines", Type: "array<object>", Desc: "命中行及上下文文本"},
					{Name: "truncated", Type: "bool", Desc: "是否因 max_files 截断"},
				},
				ResponseExample: map[string]any{
					"path":      "/20260420/23_34_52/heap",
					"filter":    "mallocgc",
					"truncated": false,
					"files": []map[string]any{
						{"path": "runtime/malloc.go", "functions": []map[string]any{{"name": "runtime.mallocgc", "flat": 2100281, "cum": 3200000}}},
					},
				},
			},
			{
				Method: "GET",
				Path:   "/api/pprof/:dir/peek",
				Desc:   "调用关系：对每个匹配正则的函数列出 callers / callees 边权重（相对该节点 cum 的占比）",
				Params: []apiParam{
					{Name: "dir", In: "path", Required: true, Desc: "profile 相对路径 base64"},
					{Name: "f", In: "query", Default: "(空)", Desc: "函数名正则；空串匹配全部（结果可能很多）"},
					{Name: "si", In: "query", Default: "-1", Desc: "SampleType 索引"},
				},
				ResponseSummary: "返回匹配节点的 callers/callees 关系与边权重",
				ResponseSchema: []apiField{
					{Name: "path/type/sample_type/sample_types/total/unit/filter", Type: "mixed", Desc: "通用元信息与过滤条件"},
					{Name: "matches", Type: "array<object>", Desc: "命中节点列表（行级）"},
					{Name: "matches[].name/file/line", Type: "mixed", Desc: "节点标识"},
					{Name: "matches[].flat/cum/flat_pct/cum_pct", Type: "mixed", Desc: "节点统计"},
					{Name: "matches[].callers|callees", Type: "array<object>", Desc: "入边/出边"},
					{Name: "edge.name/file/line/inline/weight/pct_of_cum", Type: "mixed", Desc: "边目标及占比（分母为该节点 cum）"},
				},
				ResponseExample: map[string]any{
					"filter": "mallocgc",
					"matches": []map[string]any{
						{
							"name": "runtime.mallocgc", "file": "runtime/malloc.go", "line": 1195,
							"flat": 2100281, "cum": 3200000, "flat_pct": 34.75, "cum_pct": 52.94,
							"callers": []map[string]any{{"name": "foo.alloc", "weight": 1800000, "pct_of_cum": 56.25}},
							"callees": []map[string]any{{"name": "runtime.nextFreeFast", "weight": 700000, "pct_of_cum": 21.88}},
						},
					},
				},
			},
			{
				Method: "GET",
				Path:   "/api/pprof/:dir/flame",
				Desc:   "火焰图原始数据：与 pprof Web /flamegraph 相同的 stacks + sources 模型（每样本一条栈，信息最全但体积通常更大）",
				Params: []apiParam{
					{Name: "dir", In: "path", Required: true, Desc: "profile 相对路径 base64"},
					{Name: "si", In: "query", Default: "-1", Desc: "SampleType 索引"},
					{Name: "trim_path", In: "query", Default: "", Desc: "裁剪路径前缀（与 go tool pprof -trim_path 一致，多路径用 PATH 分隔符）"},
					{Name: "source_path", In: "query", Default: "", Desc: "源码搜索路径（未设置 trim_path 时用于启发式缩短路径）"},
				},
				ResponseSummary: "返回原始 flamegraph 模型（stacks + sources），每个 sample 一条栈",
				ResponseSchema: []apiField{
					{Name: "path/type/sample_type/sample_types/total/unit/scale", Type: "mixed", Desc: "通用元信息"},
					{Name: "stacks", Type: "array<object>", Desc: "样本栈数组（条数约等于 sample 条数）"},
					{Name: "stacks[].value", Type: "int64", Desc: "该样本值"},
					{Name: "stacks[].sources", Type: "array<int>", Desc: "帧索引序列（指向 sources）"},
					{Name: "sources", Type: "array<object>", Desc: "帧字典"},
					{Name: "sources[].full_name/file_name/unique_name/inlined/display/self/color/places", Type: "mixed", Desc: "帧信息、自身值与出现位置"},
				},
				ResponseExample: map[string]any{
					"total": 6043872,
					"unit":  "bytes",
					"stacks": []map[string]any{
						{"value": 524344, "sources": []int{0, 12, 23, 31}},
					},
					"sources": []map[string]any{
						{"full_name": "root", "display": []string{"root"}, "self": 0},
						{"full_name": "http.(*conn).serve:2047", "file_name": "net/http/server.go", "inlined": false},
					},
				},
			},
			{
				Method: "GET",
				Path:   "/api/pprof/:dir/flame/layout",
				Desc:   "火焰图聚合布局：基于 /flame 合并调用前缀并给出 row/col/pct/size（更适合直接画图，但不保留每条原始 sample 栈）",
				Params: []apiParam{
					{Name: "dir", In: "path", Required: true, Desc: "profile 相对路径 base64"},
					{Name: "si", In: "query", Default: "-1", Desc: "SampleType 索引"},
					{Name: "trim_path", In: "query", Default: "", Desc: "同 /flame"},
					{Name: "source_path", In: "query", Default: "", Desc: "同 /flame"},
				},
				ResponseSummary: "返回聚合布局结果（row/col/pct/size），便于直接绘制 flame 视图",
				ResponseSchema: []apiField{
					{Name: "path/type/sample_type/sample_types/total/unit", Type: "mixed", Desc: "通用元信息"},
					{Name: "cells", Type: "array<object>", Desc: "扁平节点列表（已排序）"},
					{Name: "cells[].name/row/col/pct/value/size/size_unit/src_index/full_name", Type: "mixed", Desc: "节点展示名、位置、占比、值、可读大小与来源索引"},
					{Name: "rows", Type: "array<array<object>>", Desc: "按行分组的 cells，rows[0] 对应 row=1(root)"},
				},
				ResponseExample: map[string]any{
					"total": 6043872,
					"unit":  "bytes",
					"cells": []map[string]any{
						{"name": "root", "row": 1, "col": 1, "pct": 100, "value": 6043872, "size": "5.77", "size_unit": "MB", "src_index": 0},
						{"name": "http.(*conn).serve", "row": 2, "col": 1, "pct": 47.40, "value": 2864800, "size": "2.73", "size_unit": "MB", "src_index": 42},
					},
					"rows": []any{
						[]map[string]any{{"name": "root", "row": 1, "col": 1}},
						[]map[string]any{{"name": "http.(*conn).serve", "row": 2, "col": 1}},
					},
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

func (s *Service) APIPprofSource(c *gin.Context) {
	f := c.Query("f")
	dir := c.Param("dir")
	si, _ := strconv.Atoi(c.DefaultQuery("si", "-1"))
	margin, _ := strconv.Atoi(c.DefaultQuery("margin", "5"))
	maxFiles, _ := strconv.Atoi(c.DefaultQuery("max_files", "50"))
	sourcePath := c.Query("source_path")

	result, err := s.uc.AnalyzeSource(dir, f, si, margin, sourcePath, maxFiles)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Service) APIPprofPeek(c *gin.Context) {
	dir := c.Param("dir")
	f := c.Query("f")
	si, _ := strconv.Atoi(c.DefaultQuery("si", "-1"))

	result, err := s.uc.AnalyzePeek(dir, f, si)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Service) APIPprofFlame(c *gin.Context) {
	dir := c.Param("dir")
	si, _ := strconv.Atoi(c.DefaultQuery("si", "-1"))
	trimPath := c.Query("trim_path")
	sourcePath := c.Query("source_path")

	result, err := s.uc.AnalyzeFlame(dir, si, trimPath, sourcePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Service) APIPprofFlameLayout(c *gin.Context) {
	dir := c.Param("dir")
	si, _ := strconv.Atoi(c.DefaultQuery("si", "-1"))
	trimPath := c.Query("trim_path")
	sourcePath := c.Query("source_path")

	result, err := s.uc.AnalyzeFlameLayout(dir, si, trimPath, sourcePath)
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
