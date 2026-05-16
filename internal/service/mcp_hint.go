package service

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func resolveProfExecutable() string {
	ex, err := os.Executable()
	if err != nil {
		return ""
	}
	if abs, err := filepath.EvalSymlinks(ex); err == nil {
		ex = abs
	}
	return ex
}

func requestAPIBase(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Request.Host
	if host == "" {
		host = "127.0.0.1"
	}
	return scheme + "://" + strings.TrimSuffix(host, "/")
}

func listenPortFromRequest(c *gin.Context) string {
	host := c.Request.Host
	if i := strings.LastIndex(host, ":"); i >= 0 {
		return host[i+1:]
	}
	if c.Request.TLS != nil {
		return "443"
	}
	return "80"
}

// APIMcpHint returns MCP stdio client hints (running binary path, API base, suggested args).
func (s *Service) APIMcpHint(c *gin.Context) {
	port := listenPortFromRequest(c)
	c.JSON(200, gin.H{
		"command":  s.profBin,
		"api_base": requestAPIBase(c),
		"port":     port,
		"args":     []string{"-port", port, "mcp"},
	})
}
