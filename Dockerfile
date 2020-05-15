# syntax = docker/dockerfile:experimental

FROM golang:1.14 AS builder
WORKDIR /go/src/github.com/silvin-lubecki/docker-scan
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    make build

FROM scratch AS scan
COPY --from=builder /go/src/github.com/silvin-lubecki/docker-scan/bin/docker-scan /docker-scan