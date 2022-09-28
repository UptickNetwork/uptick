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
git checkout v0.1.0
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
Version: 0.1.0
Commit: d477e775a2596701ea215a4570e8ea9669d76edf
Checksums-Sha256:
  14ef10a820d492072f930e144aa330a21af5824fdb5c2fada442d5ec67864693  uptick_0.1.0_Darwin_x86_64.tar.gz
  576c8a13d147ca63cd1de7ee9bbabcf6f70200189cafa31b19263d3bfae37183  uptick_0.1.0_Linux_x86_64.tar.gz
  ad7bf8b237e7093478d1e2b49b3bb2a15154dc37869fa1ff98a474376bb9f07c  uptick_0.1.0_Linux_arm64.tar.gz
  d03970de154ae2438f354b5d7b25b94a22d7daa3ba7107632c7c07f188d61715  uptick_0.1.0_Darwin_arm64.tar.gz
  f8098bd12d1c9459158ec1dfd2a9a5cc4e41f960ed4de844216c65f19c803786  uptick_0.1.0_Windows_x86_64.zip
```
