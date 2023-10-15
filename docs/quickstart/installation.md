<!--
order: 1
-->

# Installation

Build and install the Uptick binaries from source or using Docker. {synopsis}

## Pre-requisites

- [Install Go 1.17.5+](https://golang.org/dl/) {prereq}
- [Install jq](https://stedolan.github.io/jq/download/) {prereq}

## Install Go

::: warning
Uptick is built using [Go](https://golang.org/dl/) version `1.17.5+`
:::

```bash
go version
```

:::tip
If the `uptickd: command not found` error message is returned, confirm that your [`GOPATH`](https://golang.org/doc/gopath_code#GOPATH) is correctly configured by running the following command:

```bash
mkdir -p $HOME/go/bin
echo "export GOPATH=$HOME/go" >> ~/.bashrc
source ~/.bashrc
echo "export GOBIN=$GOPATH/bin" >> ~/.bashrc
source ~/.bashrc
echo "export PATH=$PATH:$GOBIN" >> ~/.bashrc
source ~/.bashrc
```

:::

## Install Binaries

::: tip
The latest {{ $themeConfig.project.name }} [version](https://github.com/UptickNetwork/uptick/releases) is `{{ $themeConfig.project.binary }} {{ $themeConfig.project.latest_version }}`
:::

### GitHub

Clone and build {{ $themeConfig.project.name }} using `git`:

```bash
git clone https://github.com/UptickNetwork/uptick.git
cd uptick
make install
```

Check that the `{{ $themeConfig.project.binary }}` binaries have been successfully installed:

```bash
uptickd version
```

### Docker

You can build {{ $themeConfig.project.name }} using Docker by running:

```bash
make build-docker
```

The command above will create a docker container: `uptickhq/uptick:latest`. Now you can run `uptickd` in the container.

```bash
docker run -it -p 26657:26657 -p 26656:26656 -v ~/.uptickd/:/root/.uptickd uptickhq/uptick:latest uptickd version

# To initialize
# docker run -it -p 26657:26657 -p 26656:26656 -v ~/.uptickd/:/root/.uptickd uptickhq/uptick:latest uptickd init test-chain --chain-id test_7000-2

# To run
# docker run -it -p 26657:26657 -p 26656:26656 -v ~/.uptickd/:/root/.uptickd uptickhq/uptick:latest uptickd start
```

### Releases

You can also download a specific release available on the {{ $themeConfig.project.name }} [repository](https://github.com/UptickNetwork/uptick/releases) or via command line:

```bash
go install github.com/UptickNetwork/uptick@latest
```
