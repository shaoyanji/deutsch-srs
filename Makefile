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

.PHONY: fmt tidy test build run clean install

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
