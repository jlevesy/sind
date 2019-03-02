DIST_DIR=dist

all: build

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

test: unit_test integration_test

.phony: integration_test
integration_test:
	docker-compose \
		-f ./integration/docker-compose.yaml \
		build \
		--build-arg UID=$(shell id -u) \
		--build-arg GID=$(shell id -g)
	docker-compose \
		-f ./integration/docker-compose.yaml \
		up --exit-code-from client client
	docker-compose \
		-f ./integration/docker-compose.yaml \
		down -v

.PHONY: unit_test
unit_test:
	go test \
		-race \
		-cover \
		-timeout=5s \
		-run=$(T) \
		$(shell go list ./... | grep -v integration)

.PHONY: lint
lint:
	golangci-lint run

#
# Build targets
#

.PHONY: vendor
vendor:
	go mod download

install:
	mv ${DIST_DIR}/sind $${GOPATH}/bin/sind

build: clean dist binary

.PHONY: binary
binary:
	CGO_ENABLED=0 go build -ldflags="-s -w"  -o ${DIST_DIR}/sind ./cmd/sind

dist:
	mkdir -p ${DIST_DIR}

clean:
	rm -rf ${DIST_DIR}

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
cachedirs:
	@mkdir -p .gocache/mod
	@mkdir -p .gocache/build

