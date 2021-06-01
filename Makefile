VERSION?=$$(cat version.go | grep Version | cut -d"=" -f2 | sed 's/"//g')
GOFMT_FILES?=$$(find . -name '*.go')
PROJECT_BIN?=rce
PROJECT_SRC?=github.com/gen64/go-crud

default: test build

guard-%:
	@ if [ "${${*}}" = "" ]; then \
		echo "Environment variable $* not set"; \
		exit 1; \
	fi

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@ gofmt_files=$$(gofmt -l $(GOFMT_FILES)); \
	if [[ -n $${gofmt_files} ]]; then \
		echo "The following files fail gofmt:"; \
		echo "$${gofmt_files}"; \
		echo "Run \`make fmt\` to fix this."; \
		exit 1; \
	fi

build: guard-GOPATH
	mkdir -p $$GOPATH/bin/linux
	mkdir -p $$GOPATH/bin/darwin
	GOOS=linux GOARCH=amd64 go build -v -o $$GOPATH/bin/linux/${PROJECT_BIN} $$GOPATH/src/${PROJECT_SRC}/*.go
	GOOS=darwin GOARCH=amd64 go build -v -o $$GOPATH/bin/darwin/${PROJECT_BIN} $$GOPATH/src/${PROJECT_SRC}/*.go

test:
	go test

.NOTPARALLEL:

.PHONY: test fmt build
