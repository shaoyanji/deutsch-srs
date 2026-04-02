APP := deutsch-srs
CMD := ./cmd/$(APP)
BIN_DIR := ./bin
BIN := $(BIN_DIR)/$(APP)

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X github.com/shaoyanji/deutsch-srs/internal/version.Version=$(VERSION) \
	-X github.com/shaoyanji/deutsch-srs/internal/version.Commit=$(COMMIT) \
	-X github.com/shaoyanji/deutsch-srs/internal/version.Date=$(DATE)

.PHONY: fmt tidy test build run clean install release release-all

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy

test:
	go test ./...

build:
	mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN) $(CMD)

run:
	go run -ldflags "$(LDFLAGS)" $(CMD)

install:
	go install -ldflags "$(LDFLAGS)" $(CMD)

clean:
	rm -rf $(BIN_DIR)

# Cross-platform builds for release
release: clean
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP)-linux-amd64 $(CMD)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP)-darwin-amd64 $(CMD)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP)-darwin-arm64 $(CMD)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP)-windows-amd64.exe $(CMD)
	ls -lh $(BIN_DIR)/

# Individual platform targets
build-linux:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP)-linux-amd64 $(CMD)

build-linux-arm64:
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP)-linux-arm64 $(CMD)

build-macos: build-macos-amd64 build-macos-arm64

build-macos-amd64:
	mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP)-darwin-amd64 $(CMD)

build-macos-arm64:
	mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP)-darwin-arm64 $(CMD)

build-windows:
	mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP)-windows-amd64.exe $(CMD)

# Create release archive with checksums
release-archive: release
	cd $(BIN_DIR) && sha256sum * > checksums.txt
	cat $(BIN_DIR)/checksums.txt
