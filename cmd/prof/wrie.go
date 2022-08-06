//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/matteo-gz/prof/internal/biz"
	"github.com/matteo-gz/prof/internal/conf"
	"github.com/matteo-gz/prof/internal/data"
	"github.com/matteo-gz/prof/internal/server"
	"github.com/matteo-gz/prof/internal/service"
	"github.com/matteo-gz/prof/pkg/appx"
)

func wireApp(*conf.Bs, *conf.Server, *conf.Data, log.Logger) (*appx.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
