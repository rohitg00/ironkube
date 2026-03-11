BINARY=ironkube
VERSION=0.0.1
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X github.com/rohitg00/ironkube/cmd/ironkube/cli.Version=$(VERSION) -X github.com/rohitg00/ironkube/cmd/ironkube/cli.Commit=$(COMMIT)"

.PHONY: build test lint clean install coverage

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/ironkube

test:
	go test -race ./...

lint:
	go vet ./...

clean:
	rm -rf bin/ coverage.out coverage.html

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

install: build
	cp bin/$(BINARY) $(GOPATH)/bin/$(BINARY)
