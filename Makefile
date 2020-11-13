BINDIR      := $(CURDIR)/bin
BINNAME     ?= kubecfctl
BUILD_PLATFORMS ?=
# go option
PKG        := ./...
TAGS       :=
TESTS      := .
TESTFLAGS  :=
LDFLAGS    := -w -s
GOFLAGS    :=
SRC        := $(shell find . -type f -name '*.go' -print)

GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_TAG    = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_DIRTY  = $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")
VERSION := $(shell git describe --tags || echo $(GIT_SHA))
VERSION := $(shell echo $(VERSION) | sed -e 's/^v//g')

override LDFLAGS += -X "github.com/mudler/kubecfctl/cmd/kubecfctl.BuildTime=$(shell date -u '+%Y-%m-%d %I:%M:%S %Z')"
override LDFLAGS += -X "github.com/mudler/kubecfctl/cmd/kubecfctl.BuildCommit=$(GIT_SHA)-$(GIT_DIRTY)"

.PHONY: all
all: build

# ------------------------------------------------------------------------------
#  build

.PHONY: build
build: $(BINDIR)/$(BINNAME)

$(BINDIR)/$(BINNAME): $(SRC)
	GO111MODULE=on go build $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o '$(BINDIR)'/$(BINNAME)
	chmod +x '$(BINDIR)'/$(BINNAME)


.PHONY: deps
deps:
	go env
	GO111MODULE=off go get github.com/mitchellh/gox
	GO111MODULE=off go get github.com/onsi/ginkgo/ginkgo
	GO111MODULE=off go get github.com/onsi/gomega/...

.PHONY: multiarch-build
multiarch-build:
	CGO_ENABLED=0 gox $(BUILD_PLATFORMS) -ldflags '$(LDFLAGS)' -output="release/$(BINNAME)-$(VERSION)-{{.OS}}-{{.Arch}}"

.PHONY: clean
clean:
	@rm -rf '$(BINDIR)'

.PHONY: test
test:
	go test -v ./...

.PHONY: info
info:
	 @echo "Version:           ${VERSION}"
	 @echo "Git Tag:           ${GIT_TAG}"
	 @echo "Git Commit:        ${GIT_COMMIT}"
	 @echo "Git Tree State:    ${GIT_DIRTY}"

.PHONY: fmt
fmt:
	go fmt ./...
