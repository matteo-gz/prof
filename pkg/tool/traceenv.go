//go:build !windows

package tool

func traceEnv(s []string) []string {
	return s
}
func traceName() string {
	return "BROWSER=echo go tool trace"
}
