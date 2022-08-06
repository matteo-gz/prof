package service

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/matteo-gz/prof/internal/biz"
	"github.com/matteo-gz/prof/internal/conf"
)

type Service struct {
	uc  *biz.Usecase
	log *log.Helper
	env string
}

func NewService(c *conf.Bs, uc *biz.Usecase, logger log.Logger) *Service {
	return &Service{env: c.App.Env, uc: uc, log: log.NewHelper(logger)}
}

var ProviderSet = wire.NewSet(NewService)
