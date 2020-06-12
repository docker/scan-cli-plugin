[![Nightly Build](https://github.com/docker/docker-scan/workflows/Nightly%20Build/badge.svg)](https://github.com/docker/docker-scan/actions?query=workflow%3A%22Nightly+Build%22)

# docker-scan
Snyk CLI Plugin PoC

## Install snyk

On macOS:
```console
brew tap snyk/tap && brew install snyk
```
Other OSes:
See their [doc](https://support.snyk.io/hc/en-us/articles/360003919937-Getting-started-with-the-CLI)

## How to build and install docker scan

* You have make and go installed:
```console
$ make install
```

* You have only make and docker (of course):
```console
$ make dbuild
$ cp bin/docker-scan ~/.docker/cli-plugins
```

* You have only docker:
```console
$ @docker build . --output type=local,dest=./bin --target scan
```

Now check it's working:
```console
$ docker scan
"docker run" requires at least 1 argument.
See 'docker scan --help'.
```

## How to use it

First you need to authenticate to snyk.

* Using snyk CLI
``` console
$ snyk auth
```
It opens a browser page, you need to login, eventually using your github account.

* You can then use `docker scan DOCKER_IMAGE`: 
```console
$ docker scan hello-world

Testing hello-world...

Organization:      silvin-lubecki
Package manager:   linux
Project name:      docker-image|hello-world
Docker image:      hello-world
Licenses:          enabled

✓ Tested 0 dependencies for known issues, no vulnerable paths found.

Note that we do not currently have vulnerability data for your image.
```

* Authenticate using `docker scan --auth SNYK_AUTH_TOKEN DOCKER_IMAGE`. You need first to get your [API token](https://app.snyk.io/account)
```console
$ docker scan --auth c68dc480-27bd-45ee-9f5c-XXXXXXXXXXXX hello-world
Authenticating to Snyk using c68dc480-27bd-45ee-9f5c-XXXXXXXXXXXX

Your account has been authenticated. Snyk is now ready to be used.


Authenticated


Testing hello-world...

Organization:      silvin-lubecki
Package manager:   linux
Project name:      docker-image|hello-world
Docker image:      hello-world
Licenses:          enabled

✓ Tested 0 dependencies for known issues, no vulnerable paths found.

Note that we do not currently have vulnerability data for your image.
```

## Run end-to-end tests

You need to get a valid Snyk token and put it in the `E2E_TEST_AUTH_TOKEN` env variable.

```console
$ E2E_TEST_AUTH_TOKEN=XXXXXX make e2e
```

:warning: If you want the github actions to run on your fork, you need to define a new Github secret `E2E_TEST_AUTH_TOKEN` with your Snyk token.