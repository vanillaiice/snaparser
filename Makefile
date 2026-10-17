GO ?= go
PYTHON ?= python3

.PHONY: build test vet check release all os Windows Linux Darwin

build:
	$(GO) build -trimpath -ldflags="-s -w" -o bin/ ./cmd/snaparser

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

check: test vet
	$(GO) mod verify

# Package all six supported OS/architecture pairs with documentation and checksums.
release: check
	$(PYTHON) scripts/release.py

# Preserve the existing platform build targets (amd64).
Windows:
	mkdir -p bin/windows
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -trimpath -ldflags="-s -w" -o bin/windows/snaparser.exe ./cmd/snaparser

Linux:
	mkdir -p bin/linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath -ldflags="-s -w" -o bin/linux/snaparser-linux ./cmd/snaparser

Darwin:
	mkdir -p bin/darwin
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -trimpath -ldflags="-s -w" -o bin/darwin/snaparser-darwin ./cmd/snaparser

os: build

all: check Windows Linux Darwin
