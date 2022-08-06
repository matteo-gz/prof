//go:build windows

package tool

import "os"

func traceEnv(s []string) []string {
	s = append(s, os.Environ()...)
	s = append(s, "BROWSER=echo")
	return s
}
func traceName() string {
	return "go tool trace"
}
