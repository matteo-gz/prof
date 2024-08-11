package server

import (
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/matteo-gz/prof/internal/conf"
	"github.com/matteo-gz/prof/internal/service"
	"net/http"

	"io"
	"os"
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
	if f, err := os.Create(h.logDir + "/gin.log"); err != nil {
		return err
	} else {
		gin.DefaultWriter = io.MultiWriter(f)
	}
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
	if h.env != conf.EnvProd {
		r.LoadHTMLGlob("./web/template/*")
	}
	r.GET("/", h.srv.Index)
	r.GET("/history", h.srv.FileList)
	r.GET("/file", h.srv.File)
	r.GET(service.RoutePprofPre+"/:dir/*any", h.srv.PprofProxy)
	r.GET(service.RouteTracePre+"/:dir/*any", h.srv.TraceProxy)
	r.GET("/person/curl", h.srv.PersonCurl)
	r.GET("/bootstrap.min.css", h.srv.Css)
	r.POST("/opt/upload", h.srv.Upload)
	r.POST("/opt/run", h.srv.Run)
	r.POST("/opt/run1", h.srv.Run1)
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
