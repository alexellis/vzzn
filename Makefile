Version := $(shell git describe --tags --dirty)
GitCommit := $(shell git rev-parse HEAD)
LDFLAGS := "-s -w -X github.com/alexellis/vision/cmd.Version=$(Version) -X github.com/alexellis/vision/cmd.GitCommit=$(GitCommit)"
export GO111MODULE=on
SOURCE_DIRS = main.go cmd internal

.PHONY: all
all: gofmt test dist hash

.PHONY: test
test:
	CGO_ENABLED=0 go test $(shell go list ./... | grep -v /vendor/ | xargs echo) -cover

.PHONY: gofmt
gofmt:
	@test -z $(shell gofmt -l $(SOURCE_DIRS) ./ | grep -v vendor/ | tee /dev/stderr) || (echo "[WARN] Fix formatting issues with 'make gofmt'" && exit 1)

.PHONY: dist
dist:
	mkdir -p bin/
	rm -rf bin/vzn*
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags $(LDFLAGS) -o bin/vzn
	GOARM=7 GOARCH=arm CGO_ENABLED=0 GOOS=linux go build -ldflags $(LDFLAGS) -o bin/vzn-armhf
	GOARCH=arm64 CGO_ENABLED=0 GOOS=linux go build -ldflags $(LDFLAGS) -o bin/vzn-arm64
	CGO_ENABLED=0 GOOS=darwin go build -ldflags $(LDFLAGS) -o bin/vzn-darwin
	GOARCH=arm64 CGO_ENABLED=0 GOOS=darwin go build -ldflags $(LDFLAGS) -o bin/vzn-darwin-arm64
	GOOS=windows CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -o bin/vzn.exe

.PHONY: hash
hash:
	rm -rf bin/*.sha256 && ./hack/hashgen.sh
