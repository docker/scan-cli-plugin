export DOCKER_BUILDKIT=1

.PHONY: build
build:
	mkdir -p bin
	go build -o bin/docker-scan .

.PHONY: dbuild
dbuild:
	@docker build . \
	--output type=local,dest=./bin \
	--target scan

.PHONY: install
install: build
	cp bin/docker-scan ~/.docker/cli-plugins
