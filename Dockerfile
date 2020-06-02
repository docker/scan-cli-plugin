# syntax = docker/dockerfile:experimental
ARG GO_VERSION=1.14.3
ARG CLI_VERSION=19.03.9
ARG ALPINE_VERSION=3.12.0

FROM golang:${GO_VERSION} AS builder
WORKDIR /go/src/github.com/docker/docker-scan

# cache go vendoring
COPY go.* ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY . .

FROM builder AS build
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    make build

FROM scratch AS scan
COPY --from=build /go/src/github.com/docker/docker-scan/bin/docker-scan /docker-scan

FROM builder AS cross-build
RUN --mount=type=cache,target=/root/.cache/go-build \
    make dist

FROM scratch AS cross
COPY --from=cross-build /go/src/github.com/docker/docker-scan/dist /

FROM docker:${CLI_VERSION} AS cli

FROM alpine:${ALPINE_VERSION} AS snyk
ARG SNYK_DESKTOP_VERSION=1.332.0
ARG SNYK_USER_VERSION=1.334.0

RUN apk add -U --no-cache wget 
# install snyk desktop binary
WORKDIR /root
RUN wget https://github.com/snyk/snyk/releases/download/v${SNYK_DESKTOP_VERSION}/snyk-linux -O snyk-desktop
# install snyk user binary
RUN wget https://github.com/snyk/snyk/releases/download/v${SNYK_USER_VERSION}/snyk-linux -O snyk-user

FROM builder AS e2e
ARG SNYK_DESKTOP_VERSION=1.332.0
ENV SNYK_DESKTOP_VERSION=${SNYK_DESKTOP_VERSION}
ARG SNYK_USER_VERSION=1.334.0
ENV SNYK_USER_VERSION=${SNYK_USER_VERSION}

# install snyk binaries
COPY --from=snyk /root/snyk-desktop /root/.docker/scan/snyk
COPY --from=snyk /root/snyk-user /root/e2e/snyk
RUN chmod +x /root/.docker/scan/snyk /root/e2e/snyk
# install docker CLI
COPY --from=cli /usr/local/bin/docker /usr/local/bin/docker
# install docker-scan plugin
COPY --from=cross-build /go/src/github.com/docker/docker-scan/dist/docker-scan_linux_amd64/docker-scan/docker-scan /root/.docker/cli-plugins/docker-scan
CMD ["make", "e2e-tests"]