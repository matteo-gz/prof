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
	uc   *biz.Usecase
	log  *log.Helper
	env  string
	tmpl *template.Template
}

func NewService(c *conf.Bs, uc *biz.Usecase, logger log.Logger) *Service {
	tmpl := template.Must(template.ParseFS(web.TemplateFS, "template/*.html"))
	return &Service{
		env:  c.App.Env,
		uc:   uc,
		log:  log.NewHelper(logger),
		tmpl: tmpl,
	}
}

var ProviderSet = wire.NewSet(NewService)
