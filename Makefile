PROJECTNAME="gotools"

GOPROXY=GOPROXY=https://goproxy.cn

GOBUILD=$(GOPROXY) go build

GOMODTIDY = $(GOPROXY) go mod tidy

GOLINT=golangci-lint run

build:
	$(GOMODTIDY)
	$(GOBUILD) ./... ./...
	$(GOLINT) ./...