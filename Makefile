# Companion and art helpers. The addon itself is packaged by the BigWigs
# packager in .github/workflows/release.yml (see .pkgmeta).
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
DIST := dist
GOBUILD := CGO_ENABLED=0 go build -trimpath

.PHONY: test build macos art

test:
	cd companion && go vet -unsafeptr=false ./... && go test ./...

# Windows and Linux companions (cross-compiled from anywhere).
build:
	@mkdir -p $(DIST)
	cd companion && GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS) -H=windowsgui" -o ../$(DIST)/Slacker-Companion-Windows.exe .
	cd companion && GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o ../$(DIST)/Slacker-Companion-Linux-x64 .
	cd companion && GOOS=linux GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o ../$(DIST)/Slacker-Companion-Linux-arm64 .

# macOS app (run on a Mac).
macos:
	scripts/build-macos.sh $(VERSION)

art:
	cd companion && go run ./cmd/genart -logo ../docs/logo/logo.png -companion . ..
