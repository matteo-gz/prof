package mcpprof

import (
	"fmt"
	"os"
	"strings"

	"github.com/matteo-gz/prof/internal/conf"
)

// ResolveAPIBase returns the Prof HTTP root for MCP tool HTTP calls.
// Order: PROF_API_BASE env → (if -c set) yaml Mcp.APIBase → (if -c set) yaml Server.Port or flagPort → else flagPort only (no yaml read).
func ResolveAPIBase(flagConf, flagPort, flagEnv, flagDir, flagPprofPort, flagLog string) (string, error) {
	if v := strings.TrimSpace(os.Getenv("PROF_API_BASE")); v != "" {
		return strings.TrimRight(v, "/"), nil
	}
	if strings.TrimSpace(flagConf) == "" {
		port := strings.TrimSpace(flagPort)
		if port == "" {
			port = "8201"
		}
		return fmt.Sprintf("http://127.0.0.1:%s", port), nil
	}
	bs, err := conf.Load(flagConf, flagPort, flagEnv, flagDir, flagPprofPort, flagLog)
	if err != nil {
		return "", err
	}
	if bs.Mcp != nil && strings.TrimSpace(bs.Mcp.APIBase) != "" {
		return strings.TrimRight(strings.TrimSpace(bs.Mcp.APIBase), "/"), nil
	}
	port := ""
	if bs.Server != nil {
		port = strings.TrimSpace(bs.Server.Port)
	}
	if port == "" {
		port = strings.TrimSpace(flagPort)
	}
	if port == "" {
		port = "8201"
	}
	return fmt.Sprintf("http://127.0.0.1:%s", port), nil
}
