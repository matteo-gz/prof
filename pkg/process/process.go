//go:build !windows

package process

import (
	"bytes"
	"os/exec"
	"syscall"
)

func Kill(id, port int) {
	if pgid, err := syscall.Getpgid(id); err == nil {
		_ = syscall.Kill(-pgid, 15)
	}
}

func GetAttr() syscall.SysProcAttr {
	return syscall.SysProcAttr{
		Setpgid: true,
	}
}
func Shell(s string) *exec.Cmd {
	c := exec.Command("sh")
	in := bytes.NewBuffer(nil)
	c.Stdin = in
	in.WriteString(s)
	return c
}
