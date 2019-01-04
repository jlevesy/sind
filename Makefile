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

.PHONY: integration_test
integration_test:
	go test ./integration

.PHONY: unit_test
unit_test:
	go test -race -v -cover -timeout=5s -run=$(T) $(shell go list ./... | grep -v integration)

#
# Build targets
#

install: build
	mv ${DIST_DIR}/sind $${GOPATH}/bin/sind

build: clean dist binary

.PHONY: binary
binary:
	CGO_ENABLED=0 go build -ldflags="-s -w"  -o ${DIST_DIR}/sind ./cmd/sind

dist:
	mkdir -p ${DIST_DIR}

clean:
	rm -rf ${DIST_DIR}
