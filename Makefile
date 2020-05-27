export DOCKER_BUILDKIT=1

STATIC_FLAGS= CGO_ENABLED=0
LDFLAGS := "-s -w"
GO_BUILD = $(STATIC_FLAGS) go build -trimpath -ldflags=$(LDFLAGS)
BINARY=bin/docker-scan

.PHONY: build
build: ## Build docker-scan
	mkdir -p bin
	$(GO_BUILD) -o $(BINARY) .

.PHONY: dbuild
dbuild: ## Build docker-scan in a container
	@docker build . \
	--output type=local,dest=./bin \
	--target scan

.PHONY: cross
cross: ## Cross compile docker-scan binaries
	@GOOS=linux   GOARCH=amd64 $(GO_BUILD) -o $(BINARY)-linux-amd64 .
	@GOOS=darwin  GOARCH=amd64 $(GO_BUILD) -o $(BINARY)-darwin-amd64 .
	@GOOS=windows GOARCH=amd64 $(GO_BUILD) -o $(BINARY)-windows-amd64.exe .

.PHONY: dcross
dcross: ## Cross compile docker-scan in a container
	@docker build . \
	--output type=local,dest=./bin \
	--target cross

.PHONY: install
install: build ## Install docker-scan to your local cli-plugins directory
	cp bin/docker-scan ~/.docker/cli-plugins

help: ## Show help
	@echo Please specify a build target. The choices are:
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
