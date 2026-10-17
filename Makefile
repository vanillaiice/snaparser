GO ?= go

override VERSION := $(shell sed -n 's/^const version = "\([0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\)"$$/\1/p' cmd/snaparser/main.go)
RELEASE_VERSION ?= $(VERSION)
RELEASE_DIR ?= dist/v$(VERSION)
RELEASE_FILES := README.md LICENSE CHANGELOG.md RELEASING.md snaparser.example.toml \
	frontend/README.md frontend/assets/HTMX-LICENSE \
	$(wildcard docs/screenshots/*.png) \
	examples/jojo/README.md $(wildcard examples/jojo/json/*.json) \
	$(wildcard examples/jojo/chat_media/*)

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
	@set -eu; \
	[ -n "$(VERSION)" ] || { echo "Cannot determine the CLI version" >&2; exit 1; }; \
	[ "$(patsubst v%,%,$(RELEASE_VERSION))" = "$(VERSION)" ] || { echo "RELEASE_VERSION must match CLI version $(VERSION)" >&2; exit 1; }; \
	command -v tar >/dev/null; \
	command -v zip >/dev/null; \
	if command -v sha256sum >/dev/null; then \
		checksum() { sha256sum "$$@"; }; \
	else \
		command -v shasum >/dev/null; \
		checksum() { shasum -a 256 "$$@"; }; \
	fi; \
	release_dir="$(abspath $(RELEASE_DIR))"; \
	mkdir -p "$$release_dir"; \
	staging=$$(mktemp -d "$${TMPDIR:-/tmp}/snaparser-release.XXXXXX"); \
	trap 'rm -rf "$$staging"' 0 HUP INT TERM; \
	for system in linux darwin windows; do \
		for arch in amd64 arm64; do \
			name="snaparser_$(VERSION)_$${system}_$${arch}"; \
			package="$$staging/$$name"; \
			binary=snaparser; \
			if [ "$$system" = windows ]; then binary=snaparser.exe; fi; \
			mkdir -p "$$package"; \
			echo "Building $$name"; \
			CGO_ENABLED=0 GOOS="$$system" GOARCH="$$arch" $(GO) build -mod=readonly -trimpath -buildvcs=false -ldflags="-s -w" -o "$$package/$$binary" ./cmd/snaparser; \
			for file in $(RELEASE_FILES); do \
				mkdir -p "$$package/$$(dirname "$$file")"; \
				cp "$$file" "$$package/$$file"; \
			done; \
			if [ "$$system" = windows ]; then \
				(cd "$$staging" && zip -q -r "$$name.zip" "$$name"); \
				mv "$$staging/$$name.zip" "$$release_dir/"; \
			else \
				tar -czf "$$staging/$$name.tar.gz" -C "$$staging" "$$name"; \
				mv "$$staging/$$name.tar.gz" "$$release_dir/"; \
			fi; \
		done; \
	done; \
	(cd "$$release_dir" && checksum snaparser_$(VERSION)_*.tar.gz snaparser_$(VERSION)_*.zip) > "$$staging/SHA256SUMS"; \
	mv "$$staging/SHA256SUMS" "$$release_dir/"; \
	echo "Release archives and SHA256SUMS are ready in $$release_dir"

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
