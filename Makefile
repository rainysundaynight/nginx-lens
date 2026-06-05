# ---------- Makefile nginx-lens ----------
# Сборка, тесты и релиз Go-бинарников.

VERSION ?= $(shell grep 'Version = ' internal/version/version.go | cut -d'"' -f2)
LDFLAGS := -ldflags "-X github.com/rainysundaynight/nginx-lens/internal/version.Version=$(VERSION)"

.PHONY: build test test-coverage lint clean install

build:
	go build $(LDFLAGS) -o bin/nginx-lens ./cmd/nginx-lens
	go build $(LDFLAGS) -o bin/nginx-lens-agent ./cmd/nginx-lens-agent
	go build $(LDFLAGS) -o bin/nginx-lens-hub ./cmd/nginx-lens-hub

test:
	go test ./... -race -cover

test-coverage:
	go test ./internal/... -coverprofile=coverage.out -coverpkg=./...
	@go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run ./...

install: build
	cp bin/nginx-lens $(GOPATH)/bin/
	cp bin/nginx-lens-agent $(GOPATH)/bin/
	cp bin/nginx-lens-hub $(GOPATH)/bin/

clean:
	rm -rf bin/ dist/
