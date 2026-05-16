.PHONY: gen init start dev bu

# 程序参数写在 `--` 之后，避免 -c 等被当成 go 的选项（等同 go run main.go wire_gen.go）。
RUN_PROF = go run ./cmd/prof --

gen:
	go mod tidy
	go install github.com/google/wire/cmd/wire@latest
	cd cmd/prof && $(GOPATH)/bin/wire

init:
	cp env.yaml.example env.yaml
	go build -o prof cmd/prof/main.go cmd/prof/wire_gen.go
	./prof -c env.yaml

start:
	$(RUN_PROF) -c env.yaml

# Cursor MCP（stdio）：终端 1 先 `make start`，终端 2 再 `make dev`。
dev:
	$(RUN_PROF) -c env.yaml mcp

bu:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64  go build -o logs/prof.exe cmd/prof/main.go cmd/prof/wire_gen.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64  go build -o logs/prof_darwin cmd/prof/main.go cmd/prof/wire_gen.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build -o logs/prof_linux cmd/prof/main.go cmd/prof/wire_gen.go