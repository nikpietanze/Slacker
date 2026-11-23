# Companion and art helpers. The addon itself is packaged by the BigWigs
# packager in .github/workflows/release.yml (see .pkgmeta).
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
DIST := dist
TARGETS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: test build art

test:
	cd companion && go vet -unsafeptr=false ./... && go test ./...

build:
	@mkdir -p $(DIST)
	@for t in $(TARGETS); do \
		os=$${t%/*}; arch=$${t#*/}; ext=; flags="$(LDFLAGS)"; \
		if [ $$os = windows ]; then ext=.exe; flags="$$flags -H=windowsgui"; fi; \
		echo "building $$os/$$arch"; \
		(cd companion && CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$$flags" \
			-o ../$(DIST)/slacker-companion-$$os-$$arch$$ext .) || exit 1; \
	done

art:
	cd companion && go run ./cmd/genart -logo ../docs/logo/logo.png ..
