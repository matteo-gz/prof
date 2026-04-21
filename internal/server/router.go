package server

import (
	"io"
	"net/http"
	"os"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/matteo-gz/prof/internal/conf"
	"github.com/matteo-gz/prof/internal/service"
)

const (
	internalRoutePre = "/internal"
)

func (h *HTTPServerX) ginMode() string {
	if h.env == conf.EnvProd {
		return gin.ReleaseMode
	} else {
		return gin.DebugMode
	}
}

func (h *HTTPServerX) ginLog() error {
	h.log.Infof("log dir: %s", h.logDir)
	f, err := os.Create(h.logDir + "/gin.log")
	if err != nil {
		return err
	}
	h.ginLogFile = f
	gin.DefaultWriter = io.MultiWriter(f)
	return nil
}

func (h *HTTPServerX) router() (r *gin.Engine, err error) {
	gin.SetMode(h.ginMode())
	if err = h.ginLog(); err != nil {
		return
	}
	r = gin.Default()
	if err = r.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		return
	}
	r.GET("/", h.srv.Index)
	r.GET("/history", h.srv.FileList)
	r.GET("/file", h.srv.File)
	r.GET(service.RoutePprofPre+"/:dir/*any", h.srv.PprofProxy)
	r.GET(service.RouteTracePre+"/:dir/*any", h.srv.TraceProxy)
	r.GET(service.RoutePprofDiffPre+"/:base/:dir/*any", h.srv.PprofDiffProxy)
	r.GET("/person/curl", h.srv.PersonCurl)
	r.GET("/bootstrap.min.css", h.srv.Css)
	r.GET("/static/*filepath", h.srv.StaticFile)
	r.POST("/opt/upload", h.srv.Upload)
	r.POST("/opt/run", h.srv.Run)
	r.POST("/opt/run1", h.srv.Run1)
	api := r.Group("/api")
	{
		api.GET("/", h.srv.APIIndex)
		api.GET("/plugins", h.srv.APIPlugins)
		api.GET("/pprof/:dir/peek", h.srv.APIPprofPeek)
		api.GET("/pprof/:dir/source", h.srv.APIPprofSource)
		api.GET("/pprof/:dir/top", h.srv.APIPprofTop)
		api.GET("/pprof/:dir/flame", h.srv.APIPprofFlame)
		api.GET("/pprof/:dir/flame/layout", h.srv.APIPprofFlameLayout)
	}
	return
}

func (h *HTTPServerX) pprof() error {
	r := gin.Default()
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello world\n")
	})
	pprof.Register(r, internalRoutePre+"/debug/pprof")
	h.hs2.Handler = r
	if err2 := h.hs2.ListenAndServe(); err2 != nil && err2 != http.ErrServerClosed {
		return err2
	}
	return nil
}
