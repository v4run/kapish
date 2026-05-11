GO       ?= go
PKG      := github.com/v4run/kapish
BINDIR   := bin
BIN      := $(BINDIR)/kapish

VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS  := -X $(PKG)/internal/version.Version=$(VERSION) -X $(PKG)/internal/version.Commit=$(COMMIT)

.PHONY: all build install test lint fmt tidy clean

all: build

build:
	@mkdir -p $(BINDIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/kapish

install:
	$(GO) install -ldflags "$(LDFLAGS)" ./cmd/kapish

test:
	$(GO) test ./... -count=1

lint:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BINDIR)

.PHONY: frontend
frontend:
	cd internal/web/frontend && npm install && npm run build
