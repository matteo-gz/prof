package server

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/matteo-gz/prof/internal/conf"
	"github.com/matteo-gz/prof/internal/service"
	"net/http"
	"sync"
)

type HTTPServerX struct {
	env    string
	logDir string
	hs     *http.Server
	hs2    *http.Server
	log    *log.Helper
	srv    *service.Service
}

func (h *HTTPServerX) Start(ctx context.Context) error {
	//todo http ctx timeout
	if r, err := h.router(); err != nil {
		return err
	} else {
		h.hs.Handler = r
	}
	ch := make(chan error, 2)
	ch2 := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		ch <- h.pprof()
		return
	}()
	go func() {
		defer wg.Done()
		if err2 := h.hs.ListenAndServe(); err2 != nil && err2 != http.ErrServerClosed {
			ch <- err2
			return
		}
		ch <- nil
		return
	}()
	go func() {
		wg.Wait()
		ch2 <- struct{}{}
	}()
	for {
		select {
		case err := <-ch:
			if err != nil {
				return err
			}
		case <-ch2:
			return nil
		}
	}
}
func (h *HTTPServerX) Stop(ctx context.Context) error {
	h.log.Info("stop now")
	err := h.hs.Shutdown(ctx)
	err2 := h.hs2.Shutdown(ctx)
	if err != nil {
		return err
	}
	if err2 != nil {
		return err
	}
	return nil
}
func NewHTTPServer(c *conf.Bs, srv *service.Service, logger log.Logger) InterFace {
	hs := &http.Server{
		Addr: fmt.Sprintf(":%s", c.Server.Port),
	}
	hs2 := &http.Server{
		Addr: fmt.Sprintf(":%s", c.Server.Port2),
	}
	hsx := &HTTPServerX{
		hs:     hs,
		hs2:    hs2,
		log:    log.NewHelper(logger),
		srv:    srv,
		env:    c.App.Env,
		logDir: c.App.Log,
	}
	hsx.log.Infof("mode:%s", c.App.Env)
	hsx.log.Infof("listen:http://127.0.0.1:%s", c.Server.Port)
	hsx.log.Infof("pprof listen:http://127.0.0.1:%s", c.Server.Port2)
	return hsx
}
