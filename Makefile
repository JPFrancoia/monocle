.PHONY: build run install uninstall test check format vet lint

VERSION ?= dev

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/monocle ./cmd/monocle

run: build
	./bin/monocle

install:
	go install -ldflags "-X main.version=$(VERSION)" ./cmd/monocle

uninstall:
	rm -f $(shell go env GOPATH)/bin/monocle

test:
	go tool gotestsum -- -count=1 -cover -p 1 ./...

check:
	trivy fs --scanners vuln,secret,misconfig .

format:
	find . -name "*.go" -type f | while read -r file; do \
		golines -w --no-reformat-tags "$$file"; \
		gofmt -w "$$file"; \
	done

vet:
	go vet ./...

lint: vet
	go build ./...
