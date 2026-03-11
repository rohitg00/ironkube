BINARY=ironkube
VERSION=0.0.0
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X github.com/rohitg00/ironkube/cmd/ironkube/cli.Version=$(VERSION) -X github.com/rohitg00/ironkube/cmd/ironkube/cli.Commit=$(COMMIT)"

.PHONY: build test lint clean install

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/ironkube

test:
	go test ./... -v -race -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

install: build
	cp bin/$(BINARY) $(GOPATH)/bin/$(BINARY)
