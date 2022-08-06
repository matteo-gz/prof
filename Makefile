gen:
	go mod tidy
	go install github.com/google/wire/cmd/wire@latest
	cd cmd/prof && $(GOPATH)/bin/wire

init:
	cp env.yaml.example env.yaml
	go build -o prof cmd/prof/main.go cmd/prof/wire_gen.go
	./prof -c env.yaml

start:
	./prof -c env.yaml

bu:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64  go build -o logs/prof.exe cmd/prof/main.go cmd/prof/wire_gen.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64  go build -o logs/prof_darwin cmd/prof/main.go cmd/prof/wire_gen.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build -o logs/prof_linux cmd/prof/main.go cmd/prof/wire_gen.go