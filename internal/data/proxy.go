package data

import (
	"bytes"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/matteo-gz/prof/pkg/process"
	"github.com/matteo-gz/prof/pkg/tool"
	"os/exec"
	"sync"
	"time"
)

type Proxy struct {
	Port     int
	PortLock sync.Mutex
	CmdOut   bytes.Buffer
	CmdErr   bytes.Buffer
	useTime  int64
	Cmd      *exec.Cmd
	dir      string
	lifeTime int64
	fileType string
	log      *log.Helper
}
type detach func()

const (
	stateErr  = -1
	stateInit = 0
)

func (p *Proxy) UpdateUseTime() {
	p.useTime = time.Now().Unix()
}
func (p *Proxy) Close() {
	process.Kill(p.Cmd.Process.Pid, p.getPort())
	_ = p.Cmd.Process.Kill()
}
func (p *Proxy) kill(now int64, d detach) bool {
	if p.useTime > now-p.lifeTime || p.getPort() == stateErr {
		return false
	}
	p.PortLock.Lock()
	defer p.PortLock.Unlock()
	p.Close()
	p.setPort(stateInit)
	d()
	return true
}

func (p *Proxy) GetCanUsePort() (res int, err error) {
	switch p.getPort() {
	case stateInit:
		err = p.create()
	case stateErr:
		err = errors.New(p.getTip())
	}
	return p.getPort(), err
}
func (p *Proxy) create() error {
	p.PortLock.Lock()
	defer p.PortLock.Unlock()
	p.setPort(stateErr)
	port, err := process.GetFreePort()
	if err != nil {
		return errors.New("rand fail")
	}
	return p.start(port)
}

func (p *Proxy) cmdStd() {
	//todo
	//p.Cmd.Stdout = os.Stdout
	//p.Cmd.Stderr = os.Stdout
	p.Cmd.Stdout = &p.CmdOut
	p.Cmd.Stderr = &p.CmdErr
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
	p.setPort(port)
	return nil

}
func (p *Proxy) setPort(port int) {
	p.Port = port
}
func (p *Proxy) getPort() int {
	return p.Port
}
func (p *Proxy) getTip() string {
	return p.CmdErr.String() + "|" + p.CmdOut.String()
}
