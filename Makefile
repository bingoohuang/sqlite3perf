.PHONY: default test
all: default test

gosec:
	go get github.com/securego/gosec/cmd/gosec
sec:
	@gosec ./...
	@echo "[OK] Go security check was completed!"

init:
	export GOPROXY=https://goproxy.cn

default: install

lint:
	gofumports -w .
	gofumpt -w .
	gofmt -s -w .
	go mod tidy
	go fmt ./...
	revive .
	goimports -w .
	golangci-lint run --enable-all

install: init
	go install -ldflags="-s -w" ./...

test: init
	go test ./...

app=sqlite3perf
# https://hub.docker.com/_/golang
# docker run --rm -v "$PWD":/usr/src/myapp -v "$HOME/dockergo":/go -w /usr/src/myapp golang make docker
# docker run --rm -it -v "$PWD":/usr/src/myapp -w /usr/src/myapp golang bash
# 静态连接 glibc
docker:
	docker run --rm -v "$$PWD":/usr/src/myapp -v "$$HOME/dockergo":/go -w /usr/src/myapp golang make dockerinstall
	upx ~/dockergo/bin/${app}
	mv ~/dockergo/bin/${app}  ~/dockergo/bin/${app}-amd64-glibc2.28
	gzip ~/dockergo/bin/${app}-amd64-glibc2.28

dockerinstall:
	go install -v -x -a -ldflags '-extldflags "-static" -s -w' ./...
