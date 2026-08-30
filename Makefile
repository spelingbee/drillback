GO ?= go
BIN := bin/restored
LDFLAGS := -X github.com/spelingbee/restored/internal/cli.Version=$(VERSION) \
           -X github.com/spelingbee/restored/internal/cli.Commit=$(COMMIT) \
           -X github.com/spelingbee/restored/internal/cli.Date=$(DATE)

VERSION ?= 0.1.0-dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Every recipe except TEMPLATE, which is the skeleton a contributor copies rather than
# a recipe. Its directory name is upper-case for exactly this reason.
RECIPES := $(shell ls -d recipes/*/ 2>/dev/null | grep -v TEMPLATE | sed 's#/$$##')

.PHONY: build test test-integration lint fmt demo demo-broken demo-kuma capture-demo \
        docs recipes-index recipe-test check-generated clean

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/restored

test:
	$(GO) test ./... -race

test-integration:
	$(GO) test -tags integration ./... -timeout 30m

lint:
	gofmt -l .
	$(GO) vet ./...
	# Build-tagged files are invisible to every command above, so a signature change
	# can break the integration suite and nothing notices until CI runs it - which is
	# how `compose.Preflight` lost its second argument in session 4 and stayed broken
	# through a green `go test ./...`. This compiles them without running them.
	$(GO) vet -tags integration ./...
	golangci-lint run
	./scripts/lint-english.sh

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

# The two files derived from something else. Both are checked in, and CI fails on a
# diff, so they cannot drift from the schema and the registry they come from.
docs:
	$(GO) run ./tools/gen recipe-spec > docs/recipe-spec.md

recipes-index:
	$(GO) run ./tools/gen recipes-index > recipes/README.md
	$(GO) run ./tools/gen readme-table

check-generated: docs recipes-index
	git diff --exit-code -- docs/recipe-spec.md recipes/README.md README.md

# The round trip, against every bundled recipe. This is what recipes.yml runs, one
# recipe per matrix job; here they run in sequence.
recipe-test: build
	$(BIN) recipe test $(RECIPES)

clean:
	rm -rf bin
