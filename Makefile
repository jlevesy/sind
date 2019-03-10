DIST_DIR ?= dist

COMPOSE_FILE ?= $(CURDIR)/integration/docker-compose.yaml
export COMPOSE_FILE

.PHONY: all
all: clean build test

#
#	Release targets
#

.PHONY: release
release: clean
	goreleaser

.PHONY: dry_release
dry_release: clean
	goreleaser --skip-publish

#
# Test targets
#

.PHONY: test
test: unit_test integration_test

.phony: integration_test
integration_test:
	docker-compose \
		build \
		--build-arg UID=$(shell id -u) \
		--build-arg GID=$(shell id -g)
	docker-compose \
		up --exit-code-from client client
	docker-compose \
		down -v --remove-orphans

.PHONY: unit_test
unit_test:
	go test \
		-race \
		-cover \
		-timeout=5s \
		-run=$(T) \
		$(shell go list $(CURDIR)/... | grep -v integration)

.PHONY: lint
lint:
	golangci-lint run

#
# Build targets
#

.PHONY: vendor
vendor:
	go mod download

.PHONY: install
install:
	mv $(DIST_DIR)/sind $(GOPATH)/bin/sind

.PHONY: build
build: clean dist binary

.PHONY: binary
binary:
	CGO_ENABLED=0 go build -ldflags="-s -w"  -o $(DIST_DIR)/sind $(CURDIR)/cmd/sind

dist:
	mkdir -p $(DIST_DIR)

.PHONY: clean
clean:
	rm -rf $(DIST_DIR)

#
# Toolbox setup
#

.PHONY: toolbox
toolbox: cachedirs
	docker build \
		--build-arg=UID=$(shell id -u) \
		--build-arg=GID=$(shell id -g) \
		-t go-sind-toolbox \
		-f Dockerfile.toolbox .

.PHONY: cachedirs
cachedirs: .gocache/mod .gocache/build

.PHONY: clean-cachedirs
	rm -rf $(CURDIR)/.gocache

.gocache/mod:
	@mkdir -p $(CURDIR)/.gocache/mod

.gocache/build:
	@mkdir -p $(CURDIR)/.gocache/build
