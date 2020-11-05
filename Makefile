VERSION := $(shell git describe --exact-match --tags 2>/dev/null)
COMMIT := $(shell git rev-parse --short HEAD)

LDFLAGS := $(LDFLAGS) -X main.commit=$(COMMIT)
ifdef VERSION
	LDFLAGS += -X main.version=$(VERSION)
endif

export GO_BUILD=env GO111MODULE=on go build $(GO_ARGS) -ldflags "$(LDFLAGS) -s -w"

build:
	$(GO_BUILD) -o bin/main .

run:
	go run *.go

compile:
	# 32-Bit Systems
	# FreeBDS
	#GOOS=freebsd GOARCH=386 $(GO_BUILD) -o bin/peg-freebsd-386 .
	# MacOS
	#GOOS=darwin GOARCH=386 $(GO_BUILD) -o bin/peg-darwin-386 .
	# Linux
	#GOOS=linux GOARCH=386 $(GO_BUILD) -o bin/peg-linux-386 .
	# Windows
	#GOOS=windows GOARCH=386 $(GO_BUILD) -o bin/peg-windows-386 .

	# 64-Bit
	# FreeBDS
	#GOOS=freebsd GOARCH=amd64 $(GO_BUILD) -o bin/peg-freebsd-amd64 .
	# MacOS
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) -o bin/amd64/darwin/peg .
	upx -q -f --best bin/amd64/darwin/peg
	# Linux
	#GOOS=linux GOARCH=amd64 $(GO_BUILD) -o bin/amd64/linux/peg .
	#upx -q -f --best bin/amd64/linux/peg
	# Windows
	#GOOS=windows GOARCH=amd64 $(GO_BUILD) -o bin/peg-windows-amd64 .