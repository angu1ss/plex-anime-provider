GO_IMAGE   ?= golang:1.27-alpine
IMAGE      ?= plex-anime-provider
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

MOD_CACHE   = $(IMAGE)-gomod
BUILD_CACHE = $(IMAGE)-gobuild

GO_RUN = docker run --rm \
	-v "$(CURDIR)":/src -w /src \
	-v $(MOD_CACHE):/go/pkg/mod \
	-v $(BUILD_CACHE):/root/.cache/go-build \
	-e GOFLAGS=-buildvcs=false \
	$(GO_IMAGE)

LDFLAGS = -s -w -X main.version=$(VERSION)
OSES    = darwin linux
ARCHES  = amd64 arm64

.PHONY: help test vet fmt tidy build up down logs release clean

help: ## list available targets
	@awk 'BEGIN {FS = ":.*##"} /^[a-z-]+:.*##/ {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## run all tests
	$(GO_RUN) go test ./...

vet: ## run static analysis
	$(GO_RUN) go vet ./...

fmt: ## format sources in place
	$(GO_RUN) gofmt -l -w .

tidy: ## sync go.mod / go.sum
	$(GO_RUN) go mod tidy

build: ## build the Docker image
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

up: ## start the service via docker compose
	docker compose up -d --build

down: ## stop the service
	docker compose down

logs: ## follow service logs
	docker compose logs -f

release: ## cross-compile binaries for $(OSES) x $(ARCHES) into dist/
	$(GO_RUN) sh -ec '\
	for os in $(OSES); do for arch in $(ARCHES); do \
		echo "building dist/$(IMAGE)-$$os-$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath \
			-ldflags="$(LDFLAGS)" \
			-o dist/$(IMAGE)-$$os-$$arch ./cmd/$(IMAGE); \
	done; done'

clean: ## remove build artifacts and caches
	rm -rf dist
	docker volume rm -f $(MOD_CACHE) $(BUILD_CACHE)
