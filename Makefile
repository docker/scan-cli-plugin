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
	docker build . \
	--output type=local,dest=./bin \
	--target scan

.PHONY: cross
cross: ## Cross compile docker-scan binaries
	docker build . \
	--output type=local,dest=./dist \
	--target cross

.PHONY: install
install: build ## Install docker-scan to your local cli-plugins directory
	cp bin/docker-scan ~/.docker/cli-plugins

.PHONY: e2e-build
e2e-build: ## Build e2e docker image 
	docker build . --target e2e -t docker-scan:e2e

.PHONY: e2e
e2e: e2e-build
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock docker-scan:e2e

.PHONY: e2e-tests
e2e-tests:
	go test ./e2e

help: ## Show help
	@echo Please specify a build target. The choices are:
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
