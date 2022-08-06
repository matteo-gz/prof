//go:build windows

package process

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

func Kill(id, port int) {
	err2 := exec.Command("taskkill", "/f", "/pid", strconv.Itoa(id)).Run()
	if err2 != nil {
		fmt.Println(err2)
	}
	if pid, err := getPid(port); err == nil && pid > 0 {
		err3 := exec.Command("taskkill", "/f", "/pid", strconv.Itoa(pid)).Run()
		if err3 != nil {
			fmt.Println(err3)
		}
	}
}

func getPid(port int) (id int, err error) {
	var outBytes bytes.Buffer
	cmdStr := fmt.Sprintf("netstat -ano -p tcp | findstr 0.0.0.0:%d", port)
	cmd := exec.Command("cmd", "/c", cmdStr)
	cmd.Stdout = &outBytes
	err = cmd.Run()
	if err != nil {
		return
	}
	r := regexp.MustCompile(`\s\d+\s`).FindAllString(outBytes.String(), -1)
	if len(r) > 0 {
		id, err = strconv.Atoi(strings.TrimSpace(r[0]))
	}
	return
}

func GetAttr() syscall.SysProcAttr {
	return syscall.SysProcAttr{
		HideWindow: true,
	}
}
func Shell(s string) (c *exec.Cmd) {
	if a := strings.Split(s, " "); len(a) > 1 {
		c = exec.Command(a[0], a[1:]...)
	} else {
		c = exec.Command(s)
	}
	return
}
