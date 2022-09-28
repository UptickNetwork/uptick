<!--
order: 3
-->

# Deterministic Builds

Build the `uptickd` binary deterministically using Docker. {synopsis}

## Pre-requisites

- [Install Docker](https://docs.docker.com/get-docker/) {prereq}

## Introduction

The [Tendermint rbuilder Docker image](https://github.com/tendermint/images/tree/master/rbuilder) provides a deterministic build environment that is used to build Cosmos SDK applications. It provides a way to be reasonably sure that the executables are really built from the git source. It also makes sure that the same, tested dependencies are used and statically built into the executable.

::: tip
All the following instructions have been tested on *Ubuntu 18.04.2 LTS* with *Docker 20.10.2*.
:::

## Build with Docker

Clone `uptick`:

``` bash
git clone git@github.com:UptickNetwork/uptick.git
```

Checkout the commit, branch, or release tag you want to build (eg `v0.1.0`):

```bash
cd uptick/
git checkout <version>
```

The buildsystem supports and produces binaries for the following architectures:

* **linux/amd64**

Run the following command to launch a build for all supported architectures:

```bash
make distclean build-reproducible
```

The build system generates both the binaries and deterministic build report in the `artifacts` directory.
The `artifacts/build_report` file contains the list of the build artifacts and their respective checksums, and can be used to verify
build sanity. An example of its contents follows:

```
App: uptickd
Version: main-b64818534efefde25ef06f51c0011759a26a72ed
Commit: b64818534efefde25ef06f51c0011759a26a72ed
Checksums-Sha256:
 90c3ce170d19b6b86d31a9cf3becef68f4d232bd2ae5ec95039d2396fe0d872c  uptickd-main-b64818534efefde25ef06f51c0011759a26a72ed-linux-amd64
 d4b84ca499c847e3de5f8ac7a69ce5f436f3ae206e18de8bd355ee545d930f4f  uptickd-main-b64818534efefde25ef06f51c0011759a26a72ed.tar.gz

```
