DIST_DIR ?= dist

.PHONY: all
all: clean build test

#
#	Release targets
#

.PHONY: dry_release
dry_release: clean
	goreleaser --skip-publish

#
# Test targets
#

.PHONY: test
test: unit_test integration_test

.PHONY: integration_test
integration_test:
	go test \
		-count=1 \
		-v \
		-timeout=5m \
		-run=$(T) \
		./pkg/test

.PHONY: unit_test
unit_test:
	go test \
		-race \
		-cover \
		-timeout=5s \
		-run=$(T) \
		$(shell go list ./... | grep -v pkg/test)

.PHONY: lint
lint:
	golangci-lint run -c .golangci.yml

.PHONY: clean_docker
clean_docker:
	-docker rm -f $(shell docker ps -a -q)

#
# Build targets
#

.PHONY: download
download:
	go mod download

.PHONY: vendor
vendor:
	go mod vendor

.PHONY: install
install:
	mv $(DIST_DIR)/sind $(GOPATH)/bin/sind

.PHONY: build
build: clean dist binary

.PHONY: binary
binary:
	CGO_ENABLED=0 go build -ldflags='-s -w -X "github.com/jlevesy/sind/pkg/cli.version=custom-$(shell git rev-parse --short HEAD)" ' -o $(DIST_DIR)/sind ./cmd/sind

dist:
	mkdir -p $(DIST_DIR)

.PHONY: clean
clean:
	rm -rf $(DIST_DIR)
