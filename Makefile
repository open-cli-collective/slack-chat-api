BINARY := slack-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w \
	-X github.com/piekstra/slack-cli/internal/version.Version=$(VERSION) \
	-X github.com/piekstra/slack-cli/internal/version.Commit=$(COMMIT) \
	-X github.com/piekstra/slack-cli/internal/version.Date=$(DATE)"

DIST_DIR = dist

.PHONY: all build test test-cover test-short lint fmt deps verify clean release checksums install uninstall

all: build

build:
	go build $(LDFLAGS) -o bin/$(BINARY) .

test:
	go test -v -race ./...

test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-short:
	go test -v -short ./...

lint:
	golangci-lint run

fmt:
	go fmt ./...
	goimports -local github.com/piekstra/slack-cli -w .

deps:
	go mod download
	go mod tidy

verify:
	go mod verify

clean:
	rm -rf bin/ $(DIST_DIR)/ coverage.out coverage.html $(BINARY)

# Build for all platforms
release: clean
	mkdir -p $(DIST_DIR)

	# macOS ARM64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY) .
	tar -czvf $(DIST_DIR)/$(BINARY)_$(VERSION)_darwin_arm64.tar.gz -C $(DIST_DIR) $(BINARY)
	rm $(DIST_DIR)/$(BINARY)

	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY) .
	tar -czvf $(DIST_DIR)/$(BINARY)_$(VERSION)_darwin_amd64.tar.gz -C $(DIST_DIR) $(BINARY)
	rm $(DIST_DIR)/$(BINARY)

	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY) .
	tar -czvf $(DIST_DIR)/$(BINARY)_$(VERSION)_linux_arm64.tar.gz -C $(DIST_DIR) $(BINARY)
	rm $(DIST_DIR)/$(BINARY)

	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY) .
	tar -czvf $(DIST_DIR)/$(BINARY)_$(VERSION)_linux_amd64.tar.gz -C $(DIST_DIR) $(BINARY)
	rm $(DIST_DIR)/$(BINARY)

	@echo "Release archives created in $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/

# Generate SHA256 checksums for Homebrew formula
checksums:
	@echo "SHA256 checksums for Homebrew formula:"
	@for f in $(DIST_DIR)/*.tar.gz; do \
		echo "$$(shasum -a 256 $$f | cut -d' ' -f1)  $$(basename $$f)"; \
	done

install: build
	install -m 755 bin/$(BINARY) /usr/local/bin/

uninstall:
	rm -f /usr/local/bin/$(BINARY)
