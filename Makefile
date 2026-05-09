.PHONY: build run install uninstall test check format vet lint sync-skills skills-tarball

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

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

SKILL_NAMES := $(notdir $(patsubst %/SKILL.md,%,$(wildcard skills/*/SKILL.md)))
SKILLS_AGENTS := codex gemini

sync-skills:
	@for agent in $(SKILLS_AGENTS); do \
		rm -rf plugins/$$agent/skills; \
		mkdir -p plugins/$$agent/skills; \
		for skill in $(SKILL_NAMES); do \
			cp -r skills/$$skill plugins/$$agent/skills/$$skill; \
		done; \
	done
	@rm -rf plugins/claude/skills
	@mkdir -p plugins/claude/commands
	@cp .claude/commands/*.md plugins/claude/commands/

skills-tarball:
	mkdir -p dist
	tar -czf dist/skills.tar.gz --exclude='*.go' -C skills .
