-include .env
export

BINARY_TUI    = bin/investor
BINARY_SERVER = bin/investor-server
BINARY_LINUX  = bin/investor-linux

.PHONY: build build-tui build-server release-server deploy clean fmt vet lint test proto

proto:
	protoc \
	  --proto_path=proto \
	  --go_out=gen \
	  --go_opt=paths=source_relative \
	  --go-grpc_out=gen \
	  --go-grpc_opt=paths=source_relative \
	  investor/v1/investor.proto

fmt:
	gofmt -l -w .

vet: fmt
	go vet ./...

# requires golangci-lint: https://golangci-lint.run/welcome/install/
lint: vet
	golangci-lint run ./...

test: lint
	go test ./...

build: test build-tui build-server

build-tui:
	go build -o $(BINARY_TUI) ./cmd/investor

build-server:
	go build -o $(BINARY_SERVER) ./cmd/server

release-server:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY_LINUX) ./cmd/server

deploy: release-server
	@test -n "$(HOST)" || (echo "HOST is not set — add HOST=<ssh-hostname> to .env" && exit 1)
	rsync -az Dockerfile $(BINARY_LINUX) $(HOST):~/investor/
	ssh $(HOST) 'docker build -t investor ~/investor && docker compose -f ~/docker-compose.yml up -d investor'

clean:
	rm -rf bin/
