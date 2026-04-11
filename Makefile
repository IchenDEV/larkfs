VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test test-cover lint install clean doctor dev-mount dev-unmount

build:
	go build -ldflags "$(LDFLAGS)" -o bin/larkfs ./cmd/larkfs/

test:
	go test ./... -v -race -count=1

test-cover:
	@pkgs=$$(go list ./... | grep -v '/test$$' | grep -v '/tests/testutil$$' | paste -sd, -); \
	go test -coverpkg="$$pkgs" ./... -coverprofile=coverage.out; \
	go tool cover -func=coverage.out | tail -n 1

lint:
	golangci-lint run ./...

install: build
	cp bin/larkfs $(GOPATH)/bin/larkfs

clean:
	rm -rf bin/ dist/

doctor:
	@bin/larkfs doctor

dev-mount: build
	bin/larkfs mount /tmp/larkfs --log-level debug

dev-unmount:
	bin/larkfs unmount /tmp/larkfs
