APP=deutsch-srs

.PHONY: test build run fmt tidy

test:
	go test ./...

build:
	go build -o bin/$(APP) ./cmd/$(APP)

run:
	go run ./cmd/$(APP)

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')

tidy:
	go mod tidy
