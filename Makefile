VERSION = $(shell grep 'const version' cmd/commands/version/version.go | sed -E 's/.*"(.+)"$$/v\1/')

.PHONY: all test clean build install

GOFLAGS ?= $(GOFLAGS:)

all: install test

build:
	go build $(GOFLAGS) ./...

install:
	go get $(GOFLAGS) ./...

test: install
	go test $(GOFLAGS) ./...

bench: install
	go test -run=NONE -bench=. $(GOFLAGS) ./...

clean:
	go clean $(GOFLAGS) -i ./...

publish:
	mkdir -p bin/$(VERSION)
	cd bin/$(VERSION)
	xgo -v -x --targets="windows/*,darwin/*,linux/386,linux/amd64,linux/arm-5,linux/arm64" -out bee_$(VERSION) github.com/beego/bee
	cd ..
	ghr -u beego -r bee $(VERSION) $(VERSION)