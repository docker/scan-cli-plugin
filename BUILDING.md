# Building `docker-scan` from source

This guide is useful if you intend to contribute to `docker/scan-cli-plugin`. Thanks for your
effort. Every contribution is very appreciated.

This doc includes:
* [Build requirements](#build-requirements)
* [Build](#build)
* [Validation and Linting](#validation-and-linting)
* [Testing](#testing-docker-scan) 
* [Continuous Integration](#continuous-integration)

## Build requirements

To build the `docker-scan`, you just need Docker or Docker Desktop installed:

* [Docker (19.03 or higher)](https://www.docker.com/get-started)
* GNU Make

## Build

The whole build process is done by using a Docker container.
The images are built using BuildKit which is why you need to use at least a version `19.03` of Docker.

To build the plugin we use two Makefiles :
 - [`Makefile`](./Makefile) to run high level commands such as `build`,`cross` or `test`
 - [`builder.Makefile`](./builder.Makefile) to run container internal commands (`go build`, `gotestsum`, `golangci-lint`)

From a user point of view, you only need to use commands from the [`Makefile`](./Makefile)
```sh
make build                  # builds local platform binary
make cross                  # builds cross binaries (linux, darwin, windows)
make install                # builds a local platform binary and copy it to the `cli-plugins` directory

make all                    # lint, validate, build local plaform binary and runs unit and e2e tests
```

## Validation and Linting  

As part of the CI, two mandatory specific steps are done to check files formatting:
 - Linting: check the formatting of `Go` source files
 - Validation: check that `go.mod` and `go.sum` are up to date, check that source files contain license headers
 
You can run this validation locally

```sh
make lint                   # runs `Go` linter and checks files respect `Golang` formatting standards
make validate-go-mod        # runs a `go mod tidy` command and verifies there isn't any difference with the current `go.mod` and `go.sum` files
make validate-headers       # runs `ltag` command and checks every source files include a license header
make validate               # runs both `validate-go-mod` and `validate-headers` targets 
``` 

## Testing docker-scan

During the CI, the unit tests and end-to-end tests are executed as
part of the PR validation. As a developer you can run these tests
locally by using any of the following `Makefile` targets:
 - `make test-unit`: run all unit tests
 - `make e2e`: run all end-to-end tests
 - `make test`: run both unit and end-to-end tests

### Running specific end-to-end tests

To execute a specific end-to-end test or set of end-to-end tests you can specify
them with the E2E_TEST_NAME Makefile variable.

```console
# run the e2e test <TEST_NAME>:
make E2E_TEST_NAME=<TEST_NAME> test-e2e
```

## Continuous Integration

We use GitHub Actions to run Continuous Integration.
We currently use two different workflows:
 - [Pull Request workflow](./.github/workflows/build-pr.yml)
 - [Release/Weekly workflow](./.github/workflows/release-weekly-build.yml)
 
### Pull Request CI Workflow

This GitHub Action is divided in two steps:
 - Lint and validation checks that the `go.mod/sum` files are up to date and all source files 
 have a license header
 - Build and test builds cross platform binaries and runs unit and e2e tests (only on linux platform) 
  
### Release and weekly CI Workflow

This GitHub Action is used via cron to run each week and to trigger manually a new release.
The base behavior of both tasks is the same, it executes the following steps:
 - Create a matrix of targeted operating system
 - Install Docker CLI on the CI node
 - Set up Golang
 - Checkout code of main branch
 - Setup Golang cache
 - Download necessaries binaries (`gotestsum` ...)
 - Build and run tests (unit and e2e)
 - Upload binaries
 - Send Slack notifications

The release workflow adds these extra steps when the previous checks are OK:
 - Download previously built binaries
 - Run `ncipollo/release-action` to create a draft release
 - Send Slack notifications