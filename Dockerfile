# syntax = docker/dockerfile:experimental
ARG GO_VERSION=1.14.3
ARG CLI_VERSION=19.03.9

FROM golang:${GO_VERSION} AS builder
ARG SNYK_VERSION=1.332.0
WORKDIR /go/src/github.com/docker/docker-scan

# install NPM then Snyk
RUN curl -sL https://deb.nodesource.com/setup_13.x | bash -
RUN apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*
RUN npm install -g snyk@$SNYK_VERSION
# cache go vendoring
COPY go.* ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY . .

FROM builder AS build
RUN --mount=type=cache,target=/root/.cache/go-build \
    make build

FROM scratch AS scan
COPY --from=build /go/src/github.com/docker/docker-scan/bin/docker-scan /docker-scan

FROM builder AS cross-build
RUN --mount=type=cache,target=/root/.cache/go-build \
    make dist

FROM scratch AS cross
COPY --from=cross-build /go/src/github.com/docker/docker-scan/dist /

FROM docker:${CLI_VERSION} AS cli

FROM builder AS e2e
ARG SNYK_VERSION=1.332.0
ENV SNYK_VERSION=${SNYK_VERSION}
# install docker CLI
COPY --from=cli /usr/local/bin/docker /usr/local/bin/docker
# install docker-scan plugin
COPY --from=cross-build /go/src/github.com/docker/docker-scan/dist/docker-scan_linux_amd64/docker-scan/docker-scan /root/.docker/cli-plugins/docker-scan
CMD ["make", "e2e-tests"]