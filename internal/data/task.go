package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"path/filepath"
	"sync"
	"time"
)

type task struct {
	listLock   sync.Mutex
	list       map[string]*Proxy
	timerCheck int
	log        *log.Helper
}

func (t *task) close() {
	for _, p := range t.list {
		p.Close()
	}
}
func (t *task) Timer() {
	d := time.Duration(t.timerCheck) * time.Second
	tick := time.NewTicker(d)
	for {
		select {
		case <-tick.C:
			t.relax()
		}
	}
}

func (t *task) relax() {
	now := time.Now().Unix()
	for i, p := range t.list {
		p.kill(now, func() {
			t.Remove(i)
		})
	}
}
func (t *task) getIndex(dir string) string {
	return dir
}
func (t *task) GetProxy(dir string) (p *Proxy) {
	index := t.getIndex(dir)
	p, ok := t.list[index]
	if !ok {
		p = newProxy(filepath.Join(filepath.Split(dir)), t.log)
		t.safeOptList(func() {
			t.list[index] = p
		})
	}
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
	if err != nil {
		return
	}
	port, err = p.GetCanUsePort()
	return
}
