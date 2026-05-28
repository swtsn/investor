.PHONY: all build test vet lint clean

all: build

build: vet
	go build ./...

test: vet
	go test ./...

vet:
	go vet ./...

lint: vet
	golangci-lint run ./...

clean:
	go clean ./...
