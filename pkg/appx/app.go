package appx

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/matteo-gz/prof/internal/server"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"syscall"
)

type App struct {
	hs     server.InterFace
	logger log.Logger
	ctx    context.Context
}

// Run  start and wait for stop signal
func (app *App) Run() (err error) {
	var eg *errgroup.Group
	eg, app.ctx = errgroup.WithContext(context.Background())
	eg.Go(func() error {
		return app.hs.Start(app.ctx)
	})
	eg.Go(func() error {
		return app.sig()
	})
	return eg.Wait()
}

func (app *App) sig() (err error) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		err = app.hs.Stop(app.ctx)
		return
	case <-app.ctx.Done():
		return
	}
}

func New(logger log.Logger, hs server.InterFace) *App {
	return &App{
		hs:     hs,
		logger: logger,
	}
}
