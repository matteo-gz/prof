# Prof MCP (stdio)

Prof exposes the [Model Context Protocol](https://modelcontextprotocol.io/) over **stdio** using the official [Go SDK](https://github.com/modelcontextprotocol/go-sdk). The **`mcp` subcommand** runs in a **separate process** from the main HTTP server (tcp1 / tcp2); MCP tools proxy the existing JSON HTTP API on the main port.

On startup, `prof mcp` probes **`GET {API base}/api/`** once (2s timeout) and logs a **warning to stderr** if the web server is not reachable (stdio is unchanged). Each tool call returns a **clear error** (e.g. connection refused / timeout) if the backend is still down, so the MCP client can surface it to the model.

Chinese documentation: `pprof-ai/skills` repo → `docs/references/prof-mcp.md`.

## stdio (Cursor and CLI agents)

Start the normal Prof HTTP server, then run the **`mcp` subcommand** in a second process.

**Recommended:** reuse the same YAML as the server so the MCP client picks `Server.Port` (and optional `Mcp.APIBase`) automatically:

```bash
go build -o prof ./cmd/prof ./cmd/prof/wire_gen.go
# terminal A
./prof -c env.yaml

# terminal B — same repo, same config file
./prof -c env.yaml mcp
```

YAML (see `env.yaml.example`):

```yaml
Server:
  Port: 8201
Mcp:
  APIBase: ""   # optional; if set, overrides default http://127.0.0.1:{Server.Port}
```

Resolution order for the HTTP base MCP tools call:

1. Environment variable **`PROF_API_BASE`** (if set)
2. **`Mcp.APIBase`** from `-c` YAML (if non-empty)
3. **`http://127.0.0.1:{Server.Port}`** from YAML/`conf.Load` (falls back to `-port` flag when `-c` is omitted)

Without `-c`, **`./prof mcp` does not read any YAML**; the API base is **`http://127.0.0.1:{-port}`** (flag `-port`, default `8201`). Use `-c` when the main server’s port or `Mcp.APIBase` is only defined in a file.

### Cursor `mcpServers` example

Using the same config file as the server (adjust paths):

```json
{
  "mcpServers": {
    "prof": {
      "command": "/ABS/PATH/TO/prof",
      "args": ["-c", "/ABS/PATH/TO/env.yaml", "mcp"]
    }
  }
}
```

Only set `env.PROF_API_BASE` if you want to override YAML (e.g. Prof behind a reverse proxy).

## Tools

| Tool | Proxied HTTP |
|------|----------------|
| `prof_api_index` | `GET /api/` |
| `prof_api_plugins` | `GET /api/plugins` |
| `prof_pprof_top` | `GET /api/pprof/:dir/top` |
| `prof_pprof_source` | `GET /api/pprof/:dir/source` |
| `prof_pprof_peek` | `GET /api/pprof/:dir/peek` |
| `prof_pprof_flame` | `GET /api/pprof/:dir/flame` |
| `prof_pprof_flame_layout` | `GET /api/pprof/:dir/flame/layout` |

Tool names use underscores only (Cursor requires alphanumeric + `_`, no `.`).

The `dir` argument is the same **base64-encoded relative path** as in `/pprof/:dir/`.

Optional numeric tool arguments use **JSON omitempty**: omit the field to use the server default; include `"si": 0` when you need sample index `0` explicitly.

## Responses

Successful tool calls return the same **JSON object** as the underlying HTTP API. Errors from non-2xx HTTP or invalid JSON are returned as MCP tool errors.
