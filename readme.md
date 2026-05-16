# Prof

The web ui for golang pprof

### Requirements

- golang(pprof trace)
- graphviz

### Quick Use 
```
go install github.com/matteo-gz/prof/cmd/prof@latest
prof
```
### Start it

```
cp env.yaml.example env.yaml
go build -o prof cmd/prof/main.go cmd/prof/wire_gen.go
# start server
./prof -c env.yaml
# start server another way
./prof -port=8201 -port2=8202 env=prod -dir=./storage -log=./logs
```

## Via

- docker-compose `https://github.com/matteo-gz/prof_compose`

## How to Run

![](./doc/download.jpg)

```
# demo
# your golang service 
package main

import (
	"net/http"
	_ "net/http/pprof"
)

func main() {
	_ = http.ListenAndServe(
		"0.0.0.0:6060",
		nil,
	)
}

```

![](./doc/proxy.jpg)

### MCP (Cursor, local, stdio)

**With YAML:** `./prof -c env.yaml` then `./prof -c env.yaml mcp` (same file picks `Server.Port` / optional `Mcp.APIBase`). **Without YAML:** `./prof -port 8201` then `./prof mcp` (MCP does not read any config file; same default port). Override base URL with **`PROF_API_BASE`** if needed. See **[docs/mcp.md](./docs/mcp.md)**.

Chinese notes: `pprof-ai/skills` repo → `docs/references/prof-mcp.md`.
