# Prof

**Local pprof & trace workbench for Go.**  
Pull profiles from a running service, keep them in one place, and explore them in the browser—or let **Cursor** read the same data via MCP.

## Is this for you?

**Yes, if you:**

- Run Go services with `net/http/pprof` (or any reachable profile URL)
- Want a **simple UI** instead of juggling `curl`, files, and `go tool pprof` flags every time
- Need **saved snapshots** (heap, CPU, goroutine, trace, batch jobs) you can reopen later
- Use **Cursor** and want the agent to inspect profiles (top, source, flame, peek) on your machine

**Probably not, if you** only need a one-off `go tool pprof` on a single file and never collect from a live server.

## What you get

- **Collect** — one-shot or batch fetch from `http://your-app:6060/debug/pprof`, or upload a `.pprof` / `.trace` file
- **Browse** — history of captures under local storage; open standard pprof views (graph, flame, top, source, diff)
- **Ask AI (optional)** — connect Cursor through MCP; the UI can generate your `mcp.json` with the right binary path and port

Requires **[graphviz](https://graphviz.org/)** installed if you use the **graph** view.

## Try it in one minute

```bash
go install github.com/matteo-gz/prof/cmd/prof@latest
prof
```

Open **http://127.0.0.1:8201**, paste your app’s pprof base URL (e.g. `http://127.0.0.1:6060/debug/pprof`), pick a profile type, and submit.

**From source:**

```bash
cp env.yaml.example env.yaml
go build -o prof ./cmd/prof ./cmd/prof/wire_gen.go
./prof -c env.yaml
```

## Cursor / MCP

1. Keep Prof running (web UI on port `8201` by default).
2. In the UI, click **Use MCP** → copy the suggested config, or see **[docs/mcp.md](./docs/mcp.md)**.

## More

- **Docker:** [prof_compose](https://github.com/matteo-gz/prof_compose)
- **HTTP API & MCP tools:** [docs/mcp.md](./docs/mcp.md)

## License

Apache 2.0 — [LICENSE](./LICENSE)
