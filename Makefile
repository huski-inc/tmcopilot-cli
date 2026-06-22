APP := tmc
OPENAPI_SPEC ?= ../tmcopilot-project/backend/docs/swagger/swagger.json
VERSION ?= dev
ASSET_VERSION := $(patsubst v%,%,$(VERSION))
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/huski-inc/tmcopilot-cli/internal/version.Version=$(VERSION) \
	-X github.com/huski-inc/tmcopilot-cli/internal/version.Commit=$(COMMIT) \
	-X github.com/huski-inc/tmcopilot-cli/internal/version.Date=$(DATE)

.PHONY: test fmt vet build install clean dist openapi-sync openapi-check

test:
	go test ./...

fmt:
	gofmt -w main.go cmd internal tools

vet:
	go vet ./...

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP) .

install:
	go install -ldflags "$(LDFLAGS)" .

clean:
	rm -rf bin dist

dist:
	rm -rf dist
	mkdir -p dist/tmp
	mkdir -p dist/tmp/$(APP)-darwin-arm64
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/tmp/$(APP)-darwin-arm64/$(APP) .
	cd dist/tmp/$(APP)-darwin-arm64 && tar -czf ../../$(APP)-$(ASSET_VERSION)-darwin-arm64.tar.gz $(APP)
	mkdir -p dist/tmp/$(APP)-darwin-amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/tmp/$(APP)-darwin-amd64/$(APP) .
	cd dist/tmp/$(APP)-darwin-amd64 && tar -czf ../../$(APP)-$(ASSET_VERSION)-darwin-amd64.tar.gz $(APP)
	mkdir -p dist/tmp/$(APP)-linux-amd64
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/tmp/$(APP)-linux-amd64/$(APP) .
	cd dist/tmp/$(APP)-linux-amd64 && tar -czf ../../$(APP)-$(ASSET_VERSION)-linux-amd64.tar.gz $(APP)
	mkdir -p dist/tmp/$(APP)-linux-arm64
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/tmp/$(APP)-linux-arm64/$(APP) .
	cd dist/tmp/$(APP)-linux-arm64 && tar -czf ../../$(APP)-$(ASSET_VERSION)-linux-arm64.tar.gz $(APP)
	mkdir -p dist/tmp/$(APP)-windows-amd64
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/tmp/$(APP)-windows-amd64/$(APP).exe .
	cd dist/tmp/$(APP)-windows-amd64 && zip -q ../../$(APP)-$(ASSET_VERSION)-windows-amd64.zip $(APP).exe
	rm -rf dist/tmp
	cd dist && shasum -a 256 *.tar.gz *.zip > checksums.txt

openapi-sync:
	go run ./tools/openapi-sync --spec "$(OPENAPI_SPEC)"

openapi-check:
	go run ./tools/openapi-sync --spec "$(OPENAPI_SPEC)" --check
