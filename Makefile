.PHONY: build run install deploy uninstall test check format vet lint

VERSION ?= dev

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/monocle ./cmd/monocle

run: build
	./bin/monocle

install:
	go install -ldflags "-X main.version=$(VERSION)" ./cmd/monocle

deploy: build
	install -m 755 ./bin/monocle /home/djipey/.local/bin/monocle

uninstall:
	rm -f $(shell go env GOPATH)/bin/monocle

test:
	tmp=$$(mktemp -d); \
	trap 'rm -rf "$$tmp"' EXIT; \
	XDG_CONFIG_HOME="$$tmp" go tool gotestsum -- -count=1 -cover -p 1 ./...

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
