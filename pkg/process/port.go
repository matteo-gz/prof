package process

import (
	"net"
	"strconv"
	"sync"
	"time"
)

var portL sync.Mutex

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (port int, err error) {
	var (
		a *net.TCPAddr
		l *net.TCPListener
	)
	portL.Lock()
	defer portL.Unlock()
	a, err = net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return
	}
	l, err = net.ListenTCP("tcp", a)
	if err != nil {
		return
	}
	port = l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port, nil
}

func CheckPorts(port int) (b bool) {
	tick := time.NewTicker(time.Duration(1) * time.Second)
	af := time.After(time.Duration(5) * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-af:
			return false
		case <-tick.C:
			conn, _ := net.DialTimeout("tcp", net.JoinHostPort("", strconv.Itoa(port)), time.Second*time.Duration(5))
			if conn != nil {
				_ = conn.Close()
				return true
			}
		}
	}
}
