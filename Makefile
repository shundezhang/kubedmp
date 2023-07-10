
export GO111MODULE=on
EXECUTABLE=kubedmp
WINDOWS=$(EXECUTABLE)_windows_amd64.exe
LINUX=$(EXECUTABLE)_linux_amd64
DARWIN_AMD64=$(EXECUTABLE)_darwin_amd64
DARWIN_ARM64=$(EXECUTABLE)_darwin_arm64
VERSION=$(shell git describe --tags --always --long --dirty)
USER=$(shell git config user.name)
DATE=$(shell date)

.PHONY: test
test:
	go test ./pkg/... ./cmd/... -coverprofile cover.out

.PHONY: build
build: fmt vet
	go build -v -o bin/kubedmp -ldflags="-s -w -X 'github.com/shundezhang/kubedmp/cmd/build.Version=$(VERSION)' -X 'github.com/shundezhang/kubedmp/cmd/build.User=$(USER)' -X 'github.com/shundezhang/kubedmp/cmd/build.Time=$(DATE)'" ./cmd/main.go

.PHONY: bin
bin: fmt vet
	# go build -o bin/kubedmp github.com/shundezhang/kubedmp/cmd
	env GOOS=linux GOARCH=amd64 go build -v -o bin/$(LINUX) -ldflags="-s -w -X 'github.com/shundezhang/kubedmp/cmd/build.Version=$(VERSION)' -X 'github.com/shundezhang/kubedmp/cmd/build.User=$(USER)' -X 'github.com/shundezhang/kubedmp/cmd/build.Time=$(DATE)'" ./cmd/main.go
	tar -czvf bin/$(LINUX).tar.gz bin/$(LINUX)
	env GOOS=darwin GOARCH=amd64 go build -v -o bin/$(DARWIN_AMD64) -ldflags="-s -w -X 'github.com/shundezhang/kubedmp/cmd/build.Version=$(VERSION)' -X 'github.com/shundezhang/kubedmp/cmd/build.User=$(USER)' -X 'github.com/shundezhang/kubedmp/cmd/build.Time=$(DATE)'" ./cmd/main.go
	tar -czvf bin/$(DARWIN_AMD64).tar.gz bin/$(DARWIN_AMD64)
	env GOOS=darwin GOARCH=arm64 go build -v -o bin/$(DARWIN_ARM64) -ldflags="-s -w -X 'github.com/shundezhang/kubedmp/cmd/build.Version=$(VERSION)' -X 'github.com/shundezhang/kubedmp/cmd/build.User=$(USER)' -X 'github.com/shundezhang/kubedmp/cmd/build.Time=$(DATE)'" ./cmd/main.go
	tar -czvf bin/$(DARWIN_ARM64).tar.gz bin/$(DARWIN_ARM64)
	env GOOS=windows GOARCH=amd64 go build -v -o bin/$(WINDOWS) -ldflags="-s -w -X 'github.com/shundezhang/kubedmp/cmd/build.Version=$(VERSION)' -X 'github.com/shundezhang/kubedmp/cmd/build.User=$(USER)' -X 'github.com/shundezhang/kubedmp/cmd/build.Time=$(DATE)'" ./cmd/main.go
	zip -9 -y bin/$(EXECUTABLE)_windows_amd64.zip bin/$(WINDOWS)

.PHONY: fmt
fmt:
	go fmt ./cmd/...

.PHONY: vet
vet:
	go vet ./cmd/...

.PHONY: kubernetes-deps
kubernetes-deps:
	go get k8s.io/client-go@v11.0.0
	go get k8s.io/api@kubernetes-1.14.0
	go get k8s.io/apimachinery@kubernetes-1.14.0
	go get k8s.io/cli-runtime@kubernetes-1.14.0

.PHONY: setup
setup:
	make -C setup