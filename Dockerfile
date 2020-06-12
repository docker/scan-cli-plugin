# syntax = docker/dockerfile:experimental
ARG GO_VERSION=1.14.3
ARG CLI_VERSION=19.03.9
ARG ALPINE_VERSION=3.12.0
ARG GOLANGCI_LINT_VERSION=v1.27.0-alpine

####
# BUILDER
####
FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION} AS builder
WORKDIR /go/src/github.com/docker/docker-scan

# cache go vendoring
COPY go.* ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY . .

####
# LINT-BASE
####
FROM golangci/golangci-lint:${GOLANGCI_LINT_VERSION} AS lint-base

####
# LINT
####
FROM builder AS lint
ENV CGO_ENABLED=0
COPY --from=lint-base /usr/bin/golangci-lint /usr/bin/golangci-lint
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/.cache/golangci-lint \
    make -f builder.Makefile lint

####
# BUILD
####
FROM builder AS build
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    make -f builder.Makefile build

####
# SCAN
####
FROM scratch AS scan
COPY --from=build /go/src/github.com/docker/docker-scan/bin/docker-scan /docker-scan

####
# CROSS_BUILD
####
FROM builder AS cross-build
ARG TAG_NAME
ENV TAG_NAME=$TAG_NAME
RUN --mount=type=cache,target=/root/.cache/go-build \
    make -f builder.Makefile cross

####
# CROSS
####
FROM scratch AS cross
COPY --from=cross-build /go/src/github.com/docker/docker-scan/dist /

####
# GOTESTSUM
####
FROM alpine:${ALPINE_VERSION} AS gotestsum
ARG GOTESTSUM_VERSION=0.4.2

RUN apk add -U --no-cache wget tar
# install gotestsum
WORKDIR /root
RUN wget https://github.com/gotestyourself/gotestsum/releases/download/v${GOTESTSUM_VERSION}/gotestsum_${GOTESTSUM_VERSION}_linux_amd64.tar.gz -nv -O - | tar -xz

####
# TEST-UNIT
####
FROM builder AS test-unit
ARG TAG_NAME
ENV TAG_NAME=$TAG_NAME

# install docker-scan plugin
COPY --from=gotestsum /root/gotestsum /usr/local/bin/gotestsum
CMD ["make", "-f", "builder.Makefile", "test-unit"]

####
# CLI
####
FROM docker:${CLI_VERSION} AS cli

####
# DOWNLOAD
####
FROM golang:${GO_VERSION} AS download
ARG SNYK_DESKTOP_VERSION=1.332.0
ARG SNYK_USER_VERSION=1.334.0
COPY builder.Makefile vars.mk ./
RUN make -f builder.Makefile download

####
# E2E
####
FROM builder AS e2e
ARG SNYK_DESKTOP_VERSION=1.332.0
ENV SNYK_DESKTOP_VERSION=${SNYK_DESKTOP_VERSION}
ARG SNYK_USER_VERSION=1.334.0
ENV SNYK_USER_VERSION=${SNYK_USER_VERSION}
ARG TAG_NAME
ENV TAG_NAME=$TAG_NAME
ENV SNYK_USER_PATH="/root/e2e"
ENV SNYK_DESKTOP_PATH="/root/.docker/scan"
ENV DOCKER_CONFIG="/root/.docker"

# install snyk binaries
COPY --from=download /go/docker-config docker-config/
COPY --from=download /go/bin/gotestsum /usr/local/bin/gotestsum
# install docker CLI
COPY --from=cli /usr/local/bin/docker /usr/local/bin/docker
# install docker-scan plugin
COPY --from=cross-build /go/src/github.com/docker/docker-scan/dist/docker-scan_linux_amd64 ./bin/docker-scan
RUN chmod +x ./bin/docker-scan
CMD ["make", "-f", "builder.Makefile", "e2e"]