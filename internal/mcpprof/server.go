// Package mcpprof registers Prof JSON APIs as MCP tools (stdio only).
package mcpprof

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var httpClient = &http.Client{Timeout: 3 * time.Minute}

// RunStdio runs MCP over stdin/stdout against the given Prof HTTP root (no trailing slash).
func RunStdio(apiBase string) {
	apiBase = strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if apiBase == "" {
		apiBase = "http://127.0.0.1:8201"
	}
	warnIfProfAPIUnreachable(apiBase)
	srv := mcp.NewServer(&mcp.Implementation{Name: "prof", Version: "1.0.0"}, nil)
	RegisterTools(srv, apiBase)
	if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

// warnIfProfAPIUnreachable logs to stderr (never stdout) so stdio MCP is not corrupted.
func warnIfProfAPIUnreachable(apiBase string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	u := apiBase + "/api/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "application/json")
	c := &http.Client{Timeout: 2 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		log.Printf("mcp: [警告] 当前无法连上 Prof Web/API（%v）。Cursor 调用工具将失败，请先启动主服务；API 基址: %s", err, apiBase)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("mcp: [警告] Prof GET /api/ 返回 HTTP %d，请检查主服务是否正常；API 基址: %s", resp.StatusCode, apiBase)
	}
}

// RegisterTools attaches all Prof proxy tools to the MCP server.
func RegisterTools(s *mcp.Server, base string) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "prof_api_index",
		Description: "GET /api/ — list all JSON endpoints and parameter docs from the running Prof server",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
		return profGET(ctx, base, base+"/api/")
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "prof_api_plugins",
		Description: "GET /api/plugins — list enabled JS plugins",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
		return profGET(ctx, base, base+"/api/plugins")
	})

	type topIn struct {
		Dir string `json:"dir" jsonschema:"base64-encoded profile relative path (same as /pprof/:dir/ URL segment)"`
		N   *int   `json:"n,omitempty" jsonschema:"top N functions (server default 20 if omitted)"`
		Si  *int   `json:"si,omitempty" jsonschema:"sample type index (server default -1 = last if omitted)"`
		Cum bool   `json:"cum,omitempty" jsonschema:"if true, sort by cum instead of flat"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "prof_pprof_top",
		Description: "GET /api/pprof/:dir/top — top hot functions (flat/cum)",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in topIn) (*mcp.CallToolResult, map[string]any, error) {
		if in.Dir == "" {
			return nil, nil, fmt.Errorf("dir is required (base64 profile path)")
		}
		q := url.Values{}
		if in.N != nil {
			q.Set("n", fmt.Sprintf("%d", *in.N))
		}
		if in.Si != nil {
			q.Set("si", fmt.Sprintf("%d", *in.Si))
		}
		if in.Cum {
			q.Set("cum", "true")
		}
		u := fmt.Sprintf("%s/api/pprof/%s/top", base, url.PathEscape(in.Dir))
		if enc := q.Encode(); enc != "" {
			u += "?" + enc
		}
		return profGET(ctx, base, u)
	})

	type sourceIn struct {
		Dir        string `json:"dir" jsonschema:"base64-encoded profile relative path"`
		F          string `json:"f,omitempty" jsonschema:"function name regexp; empty matches all"`
		Si         *int   `json:"si,omitempty" jsonschema:"sample type index (server default -1 if omitted)"`
		Margin     *int   `json:"margin,omitempty" jsonschema:"context lines (server default 5 if omitted)"`
		MaxFiles   *int   `json:"max_files,omitempty" jsonschema:"max files (server default 50 if omitted)"`
		SourcePath string `json:"source_path,omitempty" jsonschema:"local source search path (PATH separator)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "prof_pprof_source",
		Description: "GET /api/pprof/:dir/source — annotated source lines with flat/cum",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in sourceIn) (*mcp.CallToolResult, map[string]any, error) {
		if in.Dir == "" {
			return nil, nil, fmt.Errorf("dir is required")
		}
		q := url.Values{}
		if in.F != "" {
			q.Set("f", in.F)
		}
		if in.Si != nil {
			q.Set("si", fmt.Sprintf("%d", *in.Si))
		}
		if in.Margin != nil {
			q.Set("margin", fmt.Sprintf("%d", *in.Margin))
		}
		if in.MaxFiles != nil {
			q.Set("max_files", fmt.Sprintf("%d", *in.MaxFiles))
		}
		if in.SourcePath != "" {
			q.Set("source_path", in.SourcePath)
		}
		u := fmt.Sprintf("%s/api/pprof/%s/source", base, url.PathEscape(in.Dir))
		if enc := q.Encode(); enc != "" {
			u += "?" + enc
		}
		return profGET(ctx, base, u)
	})

	type peekIn struct {
		Dir string `json:"dir" jsonschema:"base64-encoded profile relative path"`
		F   string `json:"f,omitempty" jsonschema:"function name regexp; empty matches all"`
		Si  *int   `json:"si,omitempty" jsonschema:"sample type index; omit for server default (-1)"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "prof_pprof_peek",
		Description: "GET /api/pprof/:dir/peek — callers/callees per matched node (line granularity)",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in peekIn) (*mcp.CallToolResult, map[string]any, error) {
		if in.Dir == "" {
			return nil, nil, fmt.Errorf("dir is required")
		}
		q := url.Values{}
		if in.F != "" {
			q.Set("f", in.F)
		}
		if in.Si != nil {
			q.Set("si", fmt.Sprintf("%d", *in.Si))
		}
		u := fmt.Sprintf("%s/api/pprof/%s/peek", base, url.PathEscape(in.Dir))
		if enc := q.Encode(); enc != "" {
			u += "?" + enc
		}
		return profGET(ctx, base, u)
	})

	type flameIn struct {
		Dir        string `json:"dir" jsonschema:"base64-encoded profile relative path"`
		Si         *int   `json:"si,omitempty" jsonschema:"sample type index; omit for server default (-1)"`
		TrimPath   string `json:"trim_path,omitempty" jsonschema:"trim path prefix (pprof -trim_path style)"`
		SourcePath string `json:"source_path,omitempty" jsonschema:"source search path for heuristic trim"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "prof_pprof_flame",
		Description: "GET /api/pprof/:dir/flame — raw stacks+sources flamegraph model (one stack per sample)",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in flameIn) (*mcp.CallToolResult, map[string]any, error) {
		if in.Dir == "" {
			return nil, nil, fmt.Errorf("dir is required")
		}
		q := url.Values{}
		if in.Si != nil {
			q.Set("si", fmt.Sprintf("%d", *in.Si))
		}
		if in.TrimPath != "" {
			q.Set("trim_path", in.TrimPath)
		}
		if in.SourcePath != "" {
			q.Set("source_path", in.SourcePath)
		}
		u := fmt.Sprintf("%s/api/pprof/%s/flame", base, url.PathEscape(in.Dir))
		if enc := q.Encode(); enc != "" {
			u += "?" + enc
		}
		return profGET(ctx, base, u)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "prof_pprof_flame_layout",
		Description: "GET /api/pprof/:dir/flame/layout — merged flame tree with row/col/pct/size",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in flameIn) (*mcp.CallToolResult, map[string]any, error) {
		if in.Dir == "" {
			return nil, nil, fmt.Errorf("dir is required")
		}
		q := url.Values{}
		if in.Si != nil {
			q.Set("si", fmt.Sprintf("%d", *in.Si))
		}
		if in.TrimPath != "" {
			q.Set("trim_path", in.TrimPath)
		}
		if in.SourcePath != "" {
			q.Set("source_path", in.SourcePath)
		}
		u := fmt.Sprintf("%s/api/pprof/%s/flame/layout", base, url.PathEscape(in.Dir))
		if enc := q.Encode(); enc != "" {
			u += "?" + enc
		}
		return profGET(ctx, base, u)
	})
}

func profGET(ctx context.Context, apiBase, rawURL string) (*mcp.CallToolResult, map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, explainProfBackendErr(apiBase, rawURL, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, rawURL, truncateBody(body, 2048))
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON from %s: %w (body: %s)", rawURL, err, truncateBody(body, 512))
	}
	return nil, out, nil
}

func truncateBody(b []byte, max int) string {
	s := string(b)
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

// explainProfBackendErr turns low-level dial errors into actionable text for the MCP client / model.
func explainProfBackendErr(apiBase, rawURL string, err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("依赖的 Prof Web/API 无响应（超时）。请求: %s。请先在另一终端启动主服务，例如 `./prof -c env.yaml` 或 `./prof -port <端口>`；当前 MCP API 基址: %q", rawURL, apiBase)
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ECONNREFUSED) {
			return fmt.Errorf("依赖的 Prof Web/API 未在该端口监听（connection refused）。请求: %s。请先启动 prof 主进程再调用 MCP；API 基址应为 %q（与主服务 HTTP 端口一致）", rawURL, apiBase)
		}
	}
	if strings.Contains(strings.ToLower(err.Error()), "connection refused") {
		return fmt.Errorf("依赖的 Prof Web/API 未启动（connection refused）。请求: %s。请先启动主服务；MCP 当前 API 基址: %q", rawURL, apiBase)
	}
	return fmt.Errorf("无法访问 Prof HTTP API（%w）。请求: %s。请确认主服务已启动且端口与 MCP 一致（API 基址 %q）", err, rawURL, apiBase)
}
