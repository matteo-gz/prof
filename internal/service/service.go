package service

import (
	"html/template"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/matteo-gz/prof/internal/biz"
	"github.com/matteo-gz/prof/internal/conf"
	"github.com/matteo-gz/prof/web"
)

type Service struct {
	uc              *biz.Usecase
	log             *log.Helper
	env             string
	port2           string
	samplingSeconds int
	deltaSeconds    int
	traceSeconds    int
	tmpl            *template.Template
}

func NewService(c *conf.Bs, uc *biz.Usecase, logger log.Logger) *Service {
	sampSec := c.App.SamplingSeconds
	if sampSec <= 0 {
		sampSec = 30
	}
	deltaSec := c.App.DeltaSeconds
	if deltaSec <= 0 {
		deltaSec = 10
	}
	traceSec := c.App.TraceSeconds
	if traceSec <= 0 {
		traceSec = 5
	}
	tmpl := template.Must(template.ParseFS(web.TemplateFS, "template/*.html"))
	return &Service{
		env:             c.App.Env,
		port2:           c.Server.Port2,
		samplingSeconds: sampSec,
		deltaSeconds:    deltaSec,
		traceSeconds:    traceSec,
		uc:              uc,
		log:             log.NewHelper(logger),
		tmpl:            tmpl,
	}
}

var ProviderSet = wire.NewSet(NewService)
