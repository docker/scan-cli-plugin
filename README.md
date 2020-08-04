![Weekly Build](https://github.com/docker/scan-cli-plugin/workflows/Release%20and%20Weekly%20Build/badge.svg)

# Docker Scan

Docker Scan is a Command Line Interface to run vulnerability detection on your Dockerfiles and Docker images.


### Table of Contents
 - **[How to use it](#how-to-use-it)**
     * **[Third Party consent](#login-and-third-party-providers)**
     * **[Scan Image or Dockerfile](#scanning)**
     * **[Provider Authentication](#provider-authentication)**
 - **[Install Docker Scan](#install-docker-scan)**
    * **[Mac & Windows](#on-macos--windows)**
    * **[Linux](#on-linux)**
 - **[Build Docker Scan](#how-to-build-docker-scan)**
 - **[Contributing](#contributing)**

## How to use it

### Login and Third Party Providers

You need to be logged into the Docker Hub in order to use the `docker scan` command.  
Docker Scan works with third party providers to detect vulnerabilities, 
the plugin will ask for your consent before sending any data to the provider.
```console
$ docker scan hello-world
? Docker Scan relies upon access to Snyk a third party provider, do you consent to proceed using Snyk? (y/N)
``` 

### Scanning

Docker Scan allows you to scan existing Docker images by name or ID. 

* You can then use `docker scan DOCKER_IMAGE`: 
```console
$  docker scan hello-world
  
  Testing hello-world...
  
  Organization:      docker-desktop-test
  Package manager:   linux
  Project name:      docker-image|hello-world
  Docker image:      hello-world
  Licenses:          enabled
  
  ✓ Tested 0 dependencies for known issues, no vulnerable paths found.
  
  Note that we do not currently have vulnerability data for your image.
```

If you want more details, you can provide the Dockerfile used to create the image
* the syntax is `docker scan -f PATH_TO_DOCKERFILE DOCKER_IMAGE`

If we apply the option to the current repository, we have:
```console
$ docker scan -f Dockerfile docker-scan:e2e
Testing docker-scan:e2e
...
✗ High severity vulnerability found in perl
  Description: Integer Overflow or Wraparound
  Info: https://snyk.io/vuln/SNYK-DEBIAN10-PERL-570802
  Introduced through: git@1:2.20.1-2+deb10u3, meta-common-packages@meta
  From: git@1:2.20.1-2+deb10u3 > perl@5.28.1-6
  From: git@1:2.20.1-2+deb10u3 > liberror-perl@0.17027-2 > perl@5.28.1-6
  From: git@1:2.20.1-2+deb10u3 > perl@5.28.1-6 > perl/perl-modules-5.28@5.28.1-6
  and 3 more...
  Introduced by your base image (golang:1.14.6)



Organization:      docker-desktop-test
Package manager:   deb
Target file:       Dockerfile
Project name:      docker-image|99138c65ebc7
Docker image:      99138c65ebc7
Base image:        golang:1.14.6
Licenses:          enabled

Tested 200 dependencies for known issues, found 157 issues.

According to our scan, you are currently using the most secure version of the selected base image
```

When using the `scan` command with the `-f` flag, you can exclude the base image (i.e.: that specified in the Dockerfile with the `FROM` directive) vulnerabilities from your report by adding the `--exclude-base` tag.
```console
$ docker scan -f Dockerfile --exclude-base docker-scan:e2e
Testing docker-scan:e2e
...
✗ Medium severity vulnerability found in libidn2/libidn2-0
  Description: Improper Input Validation
  Info: https://snyk.io/vuln/SNYK-DEBIAN10-LIBIDN2-474100
  Introduced through: iputils/iputils-ping@3:20180629-2+deb10u1, wget@1.20.1-1.1, curl@7.64.0-4+deb10u1, git@1:2.20.1-2+deb10u3
  From: iputils/iputils-ping@3:20180629-2+deb10u1 > libidn2/libidn2-0@2.0.5-1+deb10u1
  From: wget@1.20.1-1.1 > libidn2/libidn2-0@2.0.5-1+deb10u1
  From: curl@7.64.0-4+deb10u1 > curl/libcurl4@7.64.0-4+deb10u1 > libidn2/libidn2-0@2.0.5-1+deb10u1
  and 3 more...
  Introduced in your Dockerfile by 'RUN apk add -U --no-cache wget tar'



Organization:      docker-desktop-test
Package manager:   deb
Target file:       Dockerfile
Project name:      docker-image|99138c65ebc7
Docker image:      99138c65ebc7
Base image:        golang:1.14.6
Licenses:          enabled

Tested 200 dependencies for known issues, found 16 issues.
```

You can also display the scan result as a JSON output by adding the `--json` flag to the command:
```console
$ docker scan --json hello-world
{
  "vulnerabilities": [],
  "ok": true,
  "dependencyCount": 0,
  "org": "docker-desktop-test",
  "policy": "# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.\nversion: v1.19.0\nignore: {}\npatch: {}\n",
  "isPrivate": true,
  "licensesPolicy": {
    "severities": {},
    "orgLicenseRules": {
      "AGPL-1.0": {
        "licenseType": "AGPL-1.0",
        "severity": "high",
        "instructions": ""
      },
      ...
      "SimPL-2.0": {
        "licenseType": "SimPL-2.0",
        "severity": "high",
        "instructions": ""
      }
    }
  },
  "packageManager": "linux",
  "ignoreSettings": null,
  "docker": {
    "baseImageRemediation": {
      "code": "SCRATCH_BASE_IMAGE",
      "advice": [
        {
          "message": "Note that we do not currently have vulnerability data for your image.",
          "bold": true,
          "color": "yellow"
        }
      ]
    },
    "binariesVulns": {
      "issuesData": {},
      "affectedPkgs": {}
    }
  },
  "summary": "No known vulnerabilities",
  "filesystemPolicy": false,
  "uniqueCount": 0,
  "projectName": "docker-image|hello-world",
  "path": "hello-world"
}
```

If you want to see the dependency tree of your image, you can use the `--dependency-tree` flag, to display all the dependencies before the scan result
```console
$ docker-image|99138c65ebc7 @ latest
     ├─ ca-certificates @ 20200601~deb10u1
     │  └─ openssl @ 1.1.1d-0+deb10u3
     │     └─ openssl/libssl1.1 @ 1.1.1d-0+deb10u3
     ├─ curl @ 7.64.0-4+deb10u1
     │  └─ curl/libcurl4 @ 7.64.0-4+deb10u1
     │     ├─ e2fsprogs/libcom-err2 @ 1.44.5-1+deb10u3
     │     ├─ krb5/libgssapi-krb5-2 @ 1.17-3
     │     │  ├─ e2fsprogs/libcom-err2 @ 1.44.5-1+deb10u3
     │     │  ├─ krb5/libk5crypto3 @ 1.17-3
     │     │  │  └─ krb5/libkrb5support0 @ 1.17-3
     │     │  ├─ krb5/libkrb5-3 @ 1.17-3
     │     │  │  ├─ e2fsprogs/libcom-err2 @ 1.44.5-1+deb10u3
     │     │  │  ├─ krb5/libk5crypto3 @ 1.17-3
     │     │  │  ├─ krb5/libkrb5support0 @ 1.17-3
     │     │  │  └─ openssl/libssl1.1 @ 1.1.1d-0+deb10u3
     │     │  └─ krb5/libkrb5support0 @ 1.17-3
     │     ├─ libidn2/libidn2-0 @ 2.0.5-1+deb10u1
     │     │  └─ libunistring/libunistring2 @ 0.9.10-1
     │     ├─ krb5/libk5crypto3 @ 1.17-3
     │     ├─ krb5/libkrb5-3 @ 1.17-3
     │     ├─ openldap/libldap-2.4-2 @ 2.4.47+dfsg-3+deb10u2
     │     │  ├─ gnutls28/libgnutls30 @ 3.6.7-4+deb10u4
     │     │  │  ├─ nettle/libhogweed4 @ 3.4.1-1
     │     │  │  │  └─ nettle/libnettle6 @ 3.4.1-1
     │     │  │  ├─ libidn2/libidn2-0 @ 2.0.5-1+deb10u1
     │     │  │  ├─ nettle/libnettle6 @ 3.4.1-1
     │     │  │  ├─ p11-kit/libp11-kit0 @ 0.23.15-2
     │     │  │  │  └─ libffi/libffi6 @ 3.2.1-9
     │     │  │  ├─ libtasn1-6 @ 4.13-3
     │     │  │  └─ libunistring/libunistring2 @ 0.9.10-1
     │     │  ├─ cyrus-sasl2/libsasl2-2 @ 2.1.27+dfsg-1+deb10u1
     │     │  │  └─ cyrus-sasl2/libsasl2-modules-db @ 2.1.27+dfsg-1+deb10u1
     │     │  │     └─ db5.3/libdb5.3 @ 5.3.28+dfsg1-0.5
     │     │  └─ openldap/libldap-common @ 2.4.47+dfsg-3+deb10u2
     │     ├─ nghttp2/libnghttp2-14 @ 1.36.0-2+deb10u1
     │     ├─ libpsl/libpsl5 @ 0.20.2-2
     │     │  ├─ libidn2/libidn2-0 @ 2.0.5-1+deb10u1
     │     │  └─ libunistring/libunistring2 @ 0.9.10-1
     │     ├─ rtmpdump/librtmp1 @ 2.4+20151223.gitfa8646d.1-2
     │     │  ├─ gnutls28/libgnutls30 @ 3.6.7-4+deb10u4
     │     │  ├─ nettle/libhogweed4 @ 3.4.1-1
     │     │  └─ nettle/libnettle6 @ 3.4.1-1
     │     ├─ libssh2/libssh2-1 @ 1.8.0-2.1
     │     │  └─ libgcrypt20 @ 1.8.4-5
     │     └─ openssl/libssl1.1 @ 1.1.1d-0+deb10u3
     ├─ gnupg2/dirmngr @ 2.2.12-1+deb10u1
    ...

Organization:      docker-desktop-test
Package manager:   deb
Project name:      docker-image|99138c65ebc7
Docker image:      99138c65ebc7
Licenses:          enabled

Tested 200 dependencies for known issues, found 157 issues.
``` 

### Provider Authentication

If you have an existing Snyk account, you can directly use your auth token  
```console
$ docker scan --auth PROVIDER_AUTH_TOKEN
``` 

You need to get a Snyk [API token](https://app.snyk.io/account) and then use it like this
```console
$ docker scan --auth c68dc480-27bd-45ee-9f5c-XXXXXXXXXXXX

Your account has been authenticated. Snyk is now ready to be used. 
```

If you use the `auth` command without any token, you will be redirected to the Snyk website to login.

## Install Docker Scan

### On macOS & Windows:
Docker Desktop comes with Docker scan already installed.
Just try to use the plugin, open a terminal and write the following command:
```console
docker scan
Usage:	docker scan [OPTIONS] IMAGE

A tool to scan your docker image

Options:
      --accept-license    Accept to using a third party scanning provider
      --auth              Authenticate to the scan provider using an optional token, or web base token if empty
      --dependency-tree   Show dependency tree with scan results
      --exclude-base      Exclude base image from vulnerability scanning (requires --file)
  -f, --file string       Dockerfile associated with image
      --json              Output results in JSON format
      --reject-license    Reject to using a third party scanning provider
      --version           Display version of the scan plugin
```

If you get the following error message, you're not using the latest version of Docker Desktop
`docker: 'scan' is not a docker command.`

### On Linux
Download the binary from the latest release and copy it in the `cli-plugins` directory
```sh
mkdir -p ~/.docker/cli-plugins && \
curl https://github.com/docker/scan-cli-plugin/releases/download/latest/docker-scan_linux_amd64 -L -s -S -o ~/.docker/cli-plugins/docker-scan &&\
chmod +x ~/.docker/cli-plugins/docker-scan
``` 

## How to build docker scan

You'll find all the commands to build, run and test Docker Scan inside the [`BUILDING.md`](./BUILDING.md) file.

## Contributing

Want to contribute to Docker Scan? Awesome!  
First be sure to read the [Code of conduct](./CODE_OF_CONDUCT.md).   
You can find information about contributing to this project in the [`CONTRIBUTING.md`](./CONTRIBUTING.md)
