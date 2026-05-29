VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.1.0")
LDFLAGS = -s -w
BINARY = devflow-skills
MODULE = github.com/zhouhao4221/devflow-skills

.PHONY: build build-all test clean release

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

build-all:
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)_linux_amd64   .
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)_linux_arm64   .
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)_darwin_amd64  .
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)_darwin_arm64  .
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)_windows_amd64.exe .

test:
	go test ./... -v

clean:
	rm -f $(BINARY) $(BINARY)_linux_* $(BINARY)_darwin_* $(BINARY)_windows_*

release: build-all
	@echo "请手动创建 GitHub Release 并上传以下文件:"
	@ls -la $(BINARY)_*
