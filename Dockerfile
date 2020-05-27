# syntax = docker/dockerfile:experimental

FROM golang:1.14 AS builder
WORKDIR /go/src/github.com/docker/docker-scan
COPY . .

FROM builder AS build
RUN --mount=type=cache,target=/root/.cache/go-build \
    make build

FROM scratch AS scan
COPY --from=build /go/src/github.com/docker/docker-scan/bin/docker-scan /docker-scan

FROM builder AS cross-build
RUN --mount=type=cache,target=/root/.cache/go-build \
    make cross

FROM scratch AS cross
COPY --from=cross-build /go/src/github.com/docker/docker-scan/bin /