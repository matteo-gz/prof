package data

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type task struct {
	listLock   sync.Mutex
	list       map[string]*Proxy
	timerCheck int
	log        *log.Helper
	ctx        context.Context
	cancel     context.CancelFunc
}

func (t *task) close() {
	t.cancel()
	t.listLock.Lock()
	defer t.listLock.Unlock()
	for _, p := range t.list {
		p.Close()
	}
}
func (t *task) Timer() {
	d := time.Duration(t.timerCheck) * time.Second
	tick := time.NewTicker(d)
	defer tick.Stop()
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-tick.C:
			t.relax()
		}
	}
}

func (t *task) relax() {
	now := time.Now().Unix()
	var toRemove []string
	t.listLock.Lock()
	for i, p := range t.list {
		if p.useTime <= now-p.lifeTime && p.getPort() != stateErr {
			p.PortLock.Lock()
			p.closeLocked()
			p.Port = stateInit
			p.PortLock.Unlock()
			toRemove = append(toRemove, i)
		}
	}
	for _, key := range toRemove {
		delete(t.list, key)
	}
	t.listLock.Unlock()
}
func (t *task) getIndex(dir string) string {
	return dir
}
func (t *task) GetProxy(dir string) (p *Proxy) {
	index := t.getIndex(dir)
	t.listLock.Lock()
	p, ok := t.list[index]
	if !ok {
		p = newProxy(filepath.Join(filepath.Split(dir)), "", t.log)
		t.list[index] = p
	}
	t.listLock.Unlock()
	p.UpdateUseTime()
	return
}
func (t *task) Remove(index string) {
	t.safeOptList(func() {
		delete(t.list, index)
	})
}
func (t *task) safeOptList(fn func()) {
	t.listLock.Lock()
	defer t.listLock.Unlock()
	fn()
}
func (t *task) getPortByDir(dir string) (port int, err error) {
	p := t.GetProxy(dir)
	port, err = p.GetCanUsePort()
	return
}

func (t *task) getDiffIndex(base, compare string) string {
	return base + "::" + compare
}

func (t *task) GetDiffProxy(base, compare string) *Proxy {
	index := t.getDiffIndex(base, compare)
	t.listLock.Lock()
	p, ok := t.list[index]
	if !ok {
		p = newProxy(compare, base, t.log)
		t.list[index] = p
	}
	t.listLock.Unlock()
	p.UpdateUseTime()
	return p
}

func (t *task) getPortByDiff(base, compare string) (port int, err error) {
	p := t.GetDiffProxy(base, compare)
	return p.GetCanUsePort()
}
