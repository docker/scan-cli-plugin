#   Copyright 2020 Docker Inc.

#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at

#       http://www.apache.org/licenses/LICENSE-2.0

#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.
include vars.mk
export DOCKER_BUILDKIT=1

BUILD_ARGS := --build-arg GO_VERSION=$(GO_VERSION)\
	--build-arg CLI_VERSION=$(CLI_VERSION)\
	--build-arg ALPINE_VERSION=$(ALPINE_VERSION)\
	--build-arg GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) \
	--build-arg TAG_NAME=$(GIT_TAG_NAME)

.PHONY: all
all: lint validate build

.PHONY: build
build: ## Build docker-scan in a container
	docker build $(BUILD_ARGS) . \
	--output type=local,dest=./bin \
	--platform local \
	--target scan

.PHONY: cross
cross: ## Cross compile docker-scan binaries in a container
	docker build $(BUILD_ARGS) . \
	--output type=local,dest=./dist \
	--target cross

.PHONY: install
install: build ## Install docker-scan to your local cli-plugins directory
	mkdir -p $(HOME)/.docker/cli-plugins
	cp bin/$(PLATFORM_BINARY) $(HOME)/.docker/cli-plugins/$(BINARY)

.PHONY: lint
lint: ## Run the go linter
	@docker build . --target lint

.PHONY: validate-headers
validate-headers: ## Validate files license header
	docker run --rm -v $(CURDIR):/work -w /work -e LTAG_VERSION=$(LTAG_VERSION) \
	 golang:${GO_VERSION} \
	 bash -c 'go install github.com/kunalkushwaha/ltag@$(LTAG_VERSION) && ./scripts/validate/fileheader'

.PHONY: validate-go-mod
validate-go-mod: ## Validate go.mod and go.sum are up-to-date
	@docker build . --target check-go-mod

.PHONY: validate
validate: validate-go-mod validate-headers ## Validate sources

.PHONY: native-build
native-build:
	GO111MODULE=on make -f builder.Makefile build

.PHONY: help
help: ## Show help
	@echo Please specify a build target. The choices are:
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
