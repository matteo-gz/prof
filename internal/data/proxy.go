package data

import (
	"errors"
	"os/exec"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/matteo-gz/prof/pkg/process"
	"github.com/matteo-gz/prof/pkg/tool"
)

const maxCmdBufSize = 64 * 1024 // 64KB

// ringBuffer is a fixed-size circular buffer that implements io.Writer.
// When full, oldest data is silently discarded.
type ringBuffer struct {
	buf  []byte
	size int
	pos  int
	full bool
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{buf: make([]byte, size), size: size}
}

func (r *ringBuffer) Write(p []byte) (n int, err error) {
	n = len(p)
	if n >= r.size {
		copy(r.buf, p[n-r.size:])
		r.pos = 0
		r.full = true
		return
	}
	if r.pos+n <= r.size {
		copy(r.buf[r.pos:], p)
	} else {
		first := r.size - r.pos
		copy(r.buf[r.pos:], p[:first])
		copy(r.buf, p[first:])
	}
	r.pos = (r.pos + n) % r.size
	if !r.full && r.pos < n {
		r.full = true
	}
	return
}

func (r *ringBuffer) String() string {
	if !r.full {
		return string(r.buf[:r.pos])
	}
	return string(r.buf[r.pos:]) + string(r.buf[:r.pos])
}

type Proxy struct {
	Port     int
	PortLock sync.Mutex
	CmdOut   *ringBuffer
	CmdErr   *ringBuffer
	useTime  int64
	Cmd      *exec.Cmd
	dir      string
	lifeTime int64
	fileType string
	log      *log.Helper
}

const (
	stateErr  = -1
	stateInit = 0
)

func (p *Proxy) UpdateUseTime() {
	p.useTime = time.Now().Unix()
}
// closeLocked must be called with PortLock held.
func (p *Proxy) closeLocked() {
	if p.Cmd != nil && p.Cmd.Process != nil {
		process.Kill(p.Cmd.Process.Pid, p.Port)
		_ = p.Cmd.Process.Kill()
	}
}

func (p *Proxy) Close() {
	p.PortLock.Lock()
	defer p.PortLock.Unlock()
	p.closeLocked()
}

func (p *Proxy) GetCanUsePort() (res int, err error) {
	p.PortLock.Lock()
	defer p.PortLock.Unlock()
	switch p.Port {
	case stateInit:
		err = p.createLocked()
	case stateErr:
		err = errors.New(p.getTip())
	}
	return p.Port, err
}

// createLocked must be called with PortLock held.
func (p *Proxy) createLocked() error {
	p.Port = stateErr
	port, err := process.GetFreePort()
	if err != nil {
		return errors.New("rand fail")
	}
	return p.start(port)
}

func (p *Proxy) cmdStd() {
	p.Cmd.Stdout = p.CmdOut
	p.Cmd.Stderr = p.CmdErr
}
func (p *Proxy) start(port int) (err error) {
	tools := tool.NewTool(p.fileType)
	tools.Arg().SetPort(port)
	tools.Arg().SetFile(p.dir)
	cmdStr := tools.Command()
	p.log.Debug(cmdStr)
	p.Cmd = process.Shell(cmdStr)
	p.Cmd.Env = tools.Arg().AppendEnv(p.Cmd.Env)
	p.cmdStd()
	attr := process.GetAttr()
	p.Cmd.SysProcAttr = &attr
	err = p.Cmd.Start()
	if err != nil {
		return
	}
	if !process.CheckPorts(port) {
		return errors.New(p.getTip())
	}
	p.Port = port
	return nil
}
func (p *Proxy) setPort(port int) {
	p.PortLock.Lock()
	defer p.PortLock.Unlock()
	p.Port = port
}
func (p *Proxy) getPort() int {
	p.PortLock.Lock()
	defer p.PortLock.Unlock()
	return p.Port
}
func (p *Proxy) getTip() string {
	return p.CmdErr.String() + "|" + p.CmdOut.String()
}
