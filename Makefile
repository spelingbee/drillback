GO ?= go
BIN := bin/restored
LDFLAGS := -X github.com/spelingbee/restored/internal/cli.Version=$(VERSION) \
           -X github.com/spelingbee/restored/internal/cli.Commit=$(COMMIT) \
           -X github.com/spelingbee/restored/internal/cli.Date=$(DATE)

VERSION ?= 0.1.0-dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: build test test-integration lint fmt demo demo-broken demo-kuma capture-demo clean

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/restored

test:
	$(GO) test ./... -race

test-integration:
	$(GO) test -tags integration ./... -timeout 30m

lint:
	gofmt -l .
	$(GO) vet ./...
	golangci-lint run

fmt:
	gofmt -w .

demo: build
	./scripts/demo.sh

# demo-broken demonstrates a broken backup, so it exits 1 on success. The negation is
# deliberate: `make demo-broken` passes when restored correctly says RESTORE UNUSABLE.
demo-broken: build
	! ./scripts/demo-broken.sh

demo-kuma: build
	./scripts/demo-kuma.sh

capture-demo: build
	./scripts/capture-demo.sh

clean:
	rm -rf bin
