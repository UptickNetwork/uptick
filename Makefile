#!/usr/bin/make -f

BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')
VERSION := v0.4.1


# don't override user values
ifeq (,$(VERSION))
  VERSION := $(shell git describe --exact-match 2>/dev/null)
  # if VERSION is empty, then populate it with branch's name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

PACKAGES_NOSIMULATION=$(shell go list ./... | grep -v '/simulation')
PACKAGES_SIMTEST=$(shell go list ./... | grep '/simulation')

TMVERSION := $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::')
COMMIT := $(shell git log -1 --format='%H')
LEDGER_ENABLED ?= true
BINDIR ?= $(GOPATH)/bin
UPTICK_BINARY = uptickd
UPTICK_DIR = uptick
BUILDDIR ?= $(CURDIR)/build
SIMAPP = ./app
HTTPS_GIT := https://github.com/UptickNetwork/uptick.git
DOCKER := $(shell which docker)
DOCKER_BUF := $(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace bufbuild/buf
NAMESPACE := uptickhq
PROJECT := uptick
DOCKER_IMAGE := $(NAMESPACE)/$(PROJECT)
COMMIT_HASH := $(shell git rev-parse --short=7 HEAD)
DOCKER_TAG := $(COMMIT_HASH)

export GO111MODULE = on

# Default target executed when no arguments are given to make.
default_target: all

.PHONY: default_target

# process build tags

build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

ifeq (cleveldb,$(findstring cleveldb,$(COSMOS_BUILD_OPTIONS)))
  build_tags += gcc
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# process linker flags

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=uptick \
          -X github.com/cosmos/cosmos-sdk/version.AppName=$(UPTICK_BINARY) \
          -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
          -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
          -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)" \
          -X github.com/UptickNetwork/uptick/version.AppVersion=$(VERSION) \
          -X github.com/UptickNetwork/uptick/version.GitCommit=$(COMMIT) \
          -X github.com/cometbft/cometbft/version.TMCoreSemVer=$(TMVERSION)

# DB backend selection
ifeq (cleveldb,$(findstring cleveldb,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
endif
ifeq (badgerdb,$(findstring badgerdb,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=badgerdb
endif
# handle rocksdb
ifeq (rocksdb,$(findstring rocksdb,$(COSMOS_BUILD_OPTIONS)))
  CGO_ENABLED=1
  BUILD_TAGS += rocksdb
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=rocksdb
endif
# handle boltdb
ifeq (boltdb,$(findstring boltdb,$(COSMOS_BUILD_OPTIONS)))
  BUILD_TAGS += boltdb
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=boltdb
endif

ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  ldflags += -w -s
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'
# check for nostrip option
ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
endif

# # The below include contains the tools and runsim targets.
# include contrib/devtools/Makefile

###############################################################################
###                                  Build                                  ###
###############################################################################

BUILD_TARGETS := build install

build: BUILD_ARGS=-o $(BUILDDIR)/
build-linux:
	GOOS=linux GOARCH=amd64 LEDGER_ENABLED=true $(MAKE) build

$(BUILD_TARGETS): go.sum $(BUILDDIR)/
	go $@ $(BUILD_FLAGS) $(BUILD_ARGS) ./...

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

build-reproducible: go.sum
	$(DOCKER) rm latest-build || true
	$(DOCKER) run --volume=$(CURDIR):/sources:ro \
        --env TARGET_PLATFORMS='linux/amd64' \
        --env APP=uptickd \
        --env VERSION=$(VERSION) \
        --env COMMIT=$(COMMIT) \
        --env CGO_ENABLED=1 \
        --env LEDGER_ENABLED=$(LEDGER_ENABLED) \
        --name latest-build tendermintdev/rbuilder:latest
	$(DOCKER) cp -a latest-build:/home/builder/artifacts/ $(CURDIR)/


build-docker:
	# TODO replace with kaniko
	$(DOCKER) build -t ${DOCKER_IMAGE}:${DOCKER_TAG} .
	$(DOCKER) tag ${DOCKER_IMAGE}:${DOCKER_TAG} ${DOCKER_IMAGE}:latest
	# docker tag ${DOCKER_IMAGE}:${DOCKER_TAG} ${DOCKER_IMAGE}:${COMMIT_HASH}
	# update old container
	$(DOCKER) rm uptick || true
	# create a new container from the latest image
	$(DOCKER) create --name uptick -t -i ${DOCKER_IMAGE}:latest uptick
	# move the binaries to the ./build directory
	mkdir -p ./build/
	$(DOCKER) cp uptick:/usr/bin/uptickd ./build/

push-docker: build-docker
	$(DOCKER) push ${DOCKER_IMAGE}:${DOCKER_TAG}
	$(DOCKER) push ${DOCKER_IMAGE}:latest

$(MOCKS_DIR):
	mkdir -p $(MOCKS_DIR)

distclean: clean tools-clean

clean:
	rm -rf \
    $(BUILDDIR)/ \
    artifacts/ \
    tmp-swagger-gen/

all: build

build-all: tools build lint test

# `build` and `install` are BUILD_TARGETS (Makefile:130) declared as
# `$(BUILD_TARGETS): go.sum $(BUILDDIR)/`. Because BUILDDIR is `./build`, the
# target name `build` collides with the directory it writes into: once `build/`
# exists, GNU Make treats the target as already up to date and skips the compile
# entirely (rc=0, stale binary). Marking them phony forces the recipe to run.
.PHONY: distclean clean build-all build install build-linux

###############################################################################
###                          Tools & Dependencies                           ###
###############################################################################

TOOLS_DESTDIR  ?= $(GOPATH)/bin
STATIK         = $(TOOLS_DESTDIR)/statik
RUNSIM         = $(TOOLS_DESTDIR)/runsim

# Install the runsim binary.
#
# The previous recipe ran `go get github.com/cosmos/tools/cmd/runsim@master`
# from /tmp to keep go.{mod,sum} untouched. That no longer works: `go get`
# installs nothing outside a module ("go.mod file not found in current
# directory or any parent directory"), so the target and every simulation
# target that depends on it failed before running a single test. A versioned
# `go install` is the supported replacement and never touches the module of the
# directory it is run from.
runsim: $(RUNSIM)
$(RUNSIM):
	@echo "Installing runsim..."
	@GOBIN=$(TOOLS_DESTDIR) go install github.com/cosmos/tools/cmd/runsim@v1.0.0

statik: $(STATIK)
$(STATIK):
	@echo "Installing statik..."
	@GOBIN=$(TOOLS_DESTDIR) go install github.com/rakyll/statik@v0.1.6

contract-tools:
ifeq (, $(shell which stringer))
	@echo "Installing stringer..."
	@go get golang.org/x/tools/cmd/stringer
else
	@echo "stringer already installed; skipping..."
endif

ifeq (, $(shell which go-bindata))
	@echo "Installing go-bindata..."
	@go get github.com/kevinburke/go-bindata/go-bindata
else
	@echo "go-bindata already installed; skipping..."
endif

ifeq (, $(shell which gencodec))
	@echo "Installing gencodec..."
	@go get github.com/fjl/gencodec
else
	@echo "gencodec already installed; skipping..."
endif

ifeq (, $(shell which protoc-gen-go))
	@echo "Installing protoc-gen-go..."
	@go get github.com/fjl/gencodec github.com/golang/protobuf/protoc-gen-go
else
	@echo "protoc-gen-go already installed; skipping..."
endif

ifeq (, $(shell which protoc))
	@echo "Please istalling protobuf according to your OS"
	@echo "macOS: brew install protobuf"
	@echo "linux: apt-get install -f -y protobuf-compiler"
else
	@echo "protoc already installed; skipping..."
endif

ifeq (, $(shell which solcjs))
	@echo "Installing solcjs..."
	@npm install -g solc@0.5.11
else
	@echo "solcjs already installed; skipping..."
endif

docs-tools:
ifeq (, $(shell which yarn))
	@echo "Installing yarn..."
	@npm install -g yarn
else
	@echo "yarn already installed; skipping..."
endif

tools: tools-stamp
tools-stamp: contract-tools docs-tools proto-tools statik runsim
	# Create dummy file to satisfy dependency and avoid
	# rebuilding when this Makefile target is hit twice
	# in a row.
	touch $@

tools-clean:
	rm -f $(RUNSIM)
	rm -f tools-stamp

docs-tools-stamp: docs-tools
	# Create dummy file to satisfy dependency and avoid
	# rebuilding when this Makefile target is hit twice
	# in a row.
	touch $@

.PHONY: runsim statik tools contract-tools docs-tools proto-tools  tools-stamp tools-clean docs-tools-stamp

go.sum: go.mod
	echo "Ensure dependencies have not been modified ..." >&2
#	go mod verify
#	go mod tidy

###############################################################################
###                              Documentation                              ###
###############################################################################

# Regenerate the embedded Swagger bundle and fail if the result differs from
# what is committed.
#
# The previous check was written as `[ -n "$(git status --porcelain)" ]`. With a
# single `$`, `$(git status --porcelain)` is a *Make* variable reference (to an
# undefined variable), so it always expanded to the empty string and the target
# always printed "Swagger docs are in sync" and exited 0 -- even when the bundle
# was stale. It is replaced with a real shell command substitution, scoped to
# the generated package so a developer's unrelated working-tree changes do not
# trip it.
#
# `-c ""` is required for byte-identical output: the committed bundle carries no
# package comment, whereas statik v0.1.6 injects "Package statik contains static
# assets." by default. Without it, every run would report that comment as drift.
update-swagger-docs: statik
	$(BINDIR)/statik -src=client/docs/swagger-ui -dest=client/docs -f -m -c ""
	@if [ -n "$$(git status --porcelain -- client/docs/statik)" ]; then \
        echo "\033[91mSwagger docs are out of sync!!!\033[0m";\
        exit 1;\
    else \
        echo "\033[92mSwagger docs are in sync\033[0m";\
    fi
.PHONY: update-swagger-docs

godocs:
	@echo "--> Wait a few seconds and visit http://localhost:6060/pkg/github.com/UptickNetwork/uptick/types"
	godoc -http=:6060

# Start docs site at localhost:8080
docs-serve:
	@cd docs && \
	yarn && \
	yarn run serve

# Build the site into docs/.vuepress/dist
build-docs:
	@$(MAKE) docs-tools-stamp && \
	cd docs && \
	yarn && \
	yarn run build

# This builds a docs site for each branch/tag in `./docs/versions`
# and copies each site to a version prefixed path. The last entry inside
# the `versions` file will be the default root index.html.
build-docs-versioned:
	@$(MAKE) docs-tools-stamp && \
	cd docs && \
	while read -r branch path_prefix; do \
		(git checkout $${branch} && npm install && VUEPRESS_BASE="/$${path_prefix}/" npm run build) ; \
		mkdir -p ~/output/$${path_prefix} ; \
		cp -r .vuepress/dist/* ~/output/$${path_prefix}/ ; \
		cp ~/output/$${path_prefix}/index.html ~/output ; \
	done < versions ;

.PHONY: docs-serve build-docs build-docs-versioned

###############################################################################
###                           Tests & Simulation                            ###
###############################################################################

test: test-unit
test-all: test-unit test-race

# Run govulncheck and fail only on vulnerabilities not present in
# scripts/vuln-baseline.txt (known no-fix-available advisories).
# Remove IDs from the baseline once upstream publishes a fixed version.
vulncheck:
	@bash scripts/govulncheck-baseline.sh
PACKAGES_UNIT=$(shell go list ./...)
TEST_PACKAGES=./...
TEST_TARGETS := test-unit test-unit-cover test-race

# Test runs-specific rules. To add a new test target, just add
# a new rule, customise ARGS or TEST_PACKAGES ad libitum, and
# append the new rule to the TEST_TARGETS list.
test-unit: ARGS=-timeout=10m -race

test-race: ARGS=-race
test-race: TEST_PACKAGES=$(PACKAGES_NOSIMULATION)
$(TEST_TARGETS): run-tests

test-unit-cover: ARGS=-timeout=10m -race -coverprofile=coverage.txt -covermode=atomic
test-unit-cover: TEST_PACKAGES=$(PACKAGES_UNIT)

run-tests:
ifneq (,$(shell which tparse 2>/dev/null))
	go test -mod=readonly -json $(ARGS) $(EXTRA_ARGS) $(TEST_PACKAGES) | tparse
else
	go test -mod=readonly $(ARGS)  $(EXTRA_ARGS) $(TEST_PACKAGES)
endif

test-rpc:
	./scripts/integration-test-all.sh -t "rpc" -q 1 -z 1 -s 2 -m "rpc" -r "true"

test-rpc-pending:
	./scripts/integration-test-all.sh -t "pending" -q 1 -z 1 -s 2 -m "pending" -r "true"

.PHONY: run-tests test test-all test-import test-rpc $(TEST_TARGETS)

test-sim-nondeterminism:
	@echo "Running non-determinism test..."
	@go test -mod=readonly $(SIMAPP) -run TestAppStateDeterminism -Enabled=true \
		-NumBlocks=100 -BlockSize=200 -Commit=true -Period=0 -v -timeout 24h

test-sim-custom-genesis-fast:
	@echo "Running custom genesis simulation..."
	@echo "By default, ${HOME}/.$(UPTICK_DIR)/config/genesis.json will be used."
	@go test -mod=readonly $(SIMAPP) -run TestFullAppSimulation -Genesis=${HOME}/.$(UPTICK_DIR)/config/genesis.json \
		-Enabled=true -NumBlocks=100 -BlockSize=200 -Commit=true -Seed=99 -Period=5 -v -timeout 24h

test-sim-import-export: runsim
	@echo "Running application import/export simulation. This may take several minutes..."
	@$(RUNSIM) -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 5 TestAppImportExport

test-sim-after-import: runsim
	@echo "Running application simulation-after-import. This may take several minutes..."
	@$(RUNSIM) -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 5 TestAppSimulationAfterImport

test-sim-custom-genesis-multi-seed: runsim
	@echo "Running multi-seed custom genesis simulation..."
	@echo "By default, ${HOME}/.$(UPTICK_DIR)/config/genesis.json will be used."
	@$(RUNSIM) -Genesis=${HOME}/.$(UPTICK_DIR)/config/genesis.json -SimAppPkg=$(SIMAPP) -ExitOnFail 400 5 TestFullAppSimulation

test-sim-multi-seed-long: runsim
	@echo "Running long multi-seed application simulation. This may take awhile!"
	@$(RUNSIM) -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 500 50 TestFullAppSimulation

test-sim-multi-seed-short: runsim
	@echo "Running short multi-seed application simulation. This may take awhile!"
	@$(RUNSIM) -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 10 TestFullAppSimulation

# The invariant benchmark that used to live here is gone: SDK v0.53 made
# module.Manager.RegisterInvariants a no-op and x/simulation stopped asserting
# invariants, so no benchmark could assert anything. This one measures the same
# harness the simulation targets drive.
test-sim-benchmark:
	@echo "Benchmarking the application simulation..."
	@go test -mod=readonly $(SIMAPP) -benchmem -bench=BenchmarkSimulation -run=^$ \
		-Enabled=true -NumBlocks=10 -BlockSize=200 -Commit=true -Seed=57 -v -timeout 24h

# CI-sized simulation gate (audit G-05/G-06). The targets above are soak runs
# driven by runsim; this one drives the same harness with block counts small
# enough for every pull request, so determinism, export/import and
# resume-from-export stop being "only run by hand" checks.
#
# Every test here is skipped unless -Enabled=true, which is why the flag is not
# optional: without it the target would report success while doing nothing.
#
# Each line also goes through scripts/test-gate.sh, which requires one top-level
# test to have actually PASSED. -Enabled=false is not the only way to make this
# target vacuous: `go test` also exits 0 when -run matches no test at all, which
# is what a rename would produce.
test-sim-ci:
	@echo "Running the CI-sized simulation gate..."
	@./scripts/test-gate.sh 1 -- -mod=readonly $(SIMAPP) -run TestAppStateDeterminism -Enabled=true \
		-NumBlocks=5 -BlockSize=4 -Commit=true -Seed=1 -Period=0 -v -timeout 20m
	@./scripts/test-gate.sh 1 -- -mod=readonly $(SIMAPP) -run TestAppImportExport -Enabled=true \
		-NumBlocks=5 -BlockSize=4 -Commit=true -Seed=1 -Period=0 -v -timeout 20m
	@./scripts/test-gate.sh 1 -- -mod=readonly $(SIMAPP) -run TestAppSimulationAfterImport -Enabled=true \
		-NumBlocks=4 -BlockSize=3 -Commit=true -Seed=1 -Period=0 -v -timeout 20m
	@./scripts/test-gate.sh 1 -- -mod=readonly $(SIMAPP) -run TestFullAppSimulation -Enabled=true \
		-NumBlocks=5 -BlockSize=4 -Commit=true -Seed=1 -Period=0 -v -timeout 20m
.PHONY: test-sim-ci

###############################################################################
###                             End-to-end tests                             ###
###############################################################################

# The e2e suite needs a real node: it signs Keplr-style transactions and probes
# the EVM JSON-RPC endpoint. scripts/e2e-localnet.sh owns the node lifecycle so
# the same command works locally and in CI.
#
# UPTICK_E2E_STRICT=1 makes an unreachable node a failure instead of a skip -
# otherwise this target could pass without ever talking to a chain.
#
# On top of that, every e2e run below goes through scripts/test-gate.sh and must
# actually pass E2E_MIN_TESTS top-level tests. UPTICK_E2E_STRICT only covers a
# node that cannot be reached; a deleted test file, or a -build-tag change that
# excludes the file, would still exit 0. Update this count when the suite
# changes on purpose.
E2E_MIN_TESTS = 5

e2e-localnet-start:
	@./scripts/e2e-localnet.sh start

e2e-localnet-stop:
	@./scripts/e2e-localnet.sh stop

# Start a node, run the e2e suite against it with strict mode on, always stop
# the node.
#
# Node, test and teardown are joined into ONE shell on purpose. Make runs each
# recipe line in its own shell, and a job runner reaps a step's process tree
# when that shell exits - so `start` in its own recipe line reliably leaves the
# test talking to a node that is already dead. Keeping it a single shell is
# also what makes `exit $$status` meaningful: the failure of the test, not of
# the teardown, decides the target's exit code.
test-e2e-localnet:
	@status=0; \
	./scripts/e2e-localnet.sh start || status=$$?; \
	if [ $$status -eq 0 ]; then \
		UPTICK_E2E_STRICT=1 ./scripts/test-gate.sh $(E2E_MIN_TESTS) -- -mod=readonly ./tests/e2e/... -count=1 -v -timeout 15m || status=$$?; \
	fi; \
	./scripts/e2e-localnet.sh stop || true; \
	exit $$status
.PHONY: test-e2e-localnet e2e-localnet-start e2e-localnet-stop

# Run the e2e suite against an already running node (see e2e-localnet-start).
test-e2e:
	@./scripts/test-gate.sh $(E2E_MIN_TESTS) -- -mod=readonly ./tests/e2e/... -count=1 -v -timeout 15m
.PHONY: test-e2e

.PHONY: \
test-sim-nondeterminism \
test-sim-custom-genesis-fast \
test-sim-import-export \
test-sim-after-import \
test-sim-custom-genesis-multi-seed \
test-sim-multi-seed-short \
test-sim-multi-seed-long \
test-sim-benchmark

benchmark:
	@go test -mod=readonly -bench=. $(PACKAGES_NOSIMULATION)
.PHONY: benchmark

###############################################################################
###                                Linting                                  ###
###############################################################################

lint:
	golangci-lint run

lint-contracts:
	@cd contracts && \
	npm i && \
	npm run lint

lint-fix:
	golangci-lint run --fix --issues-exit-code=0

lint-fix-contracts:
	@cd contracts && \
	npm i && \
	npm run lint-fix

.PHONY: lint lint-fix

format:
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" -not -name '*.pb.go' | xargs gofmt -w -s
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" -not -name '*.pb.go' | xargs misspell -w
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" -not -name '*.pb.go' | xargs goimports -w -local github.com/UptickNetwork/uptick
.PHONY: format


###############################################################################
###                                Protobuf                                 ###
###############################################################################
protoVer=0.14.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

proto-all: proto-format proto-lint proto-gen

proto-gen:
	@echo "Generating Protobuf files"
	@$(protoImage) sh ./scripts/protocgen.sh

proto-swagger-gen:
	@echo "Generating Protobuf Swagger"
	@$(protoImage) sh ./scripts/protoc-swagger-gen.sh

proto-format:
	@$(protoImage) buf format --write proto/

proto-lint:
	@$(protoImage) buf lint --error-format=json


drone-generate:
	drone starlark --format --target .drone.star.yml




###############################################################################
###                                Localnet                                 ###
###############################################################################

# Build image for a local testnet
localnet-build:
	@$(MAKE) -C networks/local

# Start a 4-node testnet locally
localnet-start: localnet-stop
ifeq ($(OS),Windows_NT)
	mkdir localnet-setup &
	@$(MAKE) localnet-build

	IF not exist "build/node0/$(UPTICK_BINARY)/config/genesis.json" docker run --rm -v $(CURDIR)/build\uptick\Z uptick/node:v0.2 "./uptickd testnet --v 4 -o /uptick --keyring-backend=test --starting-ip-address 192.167.10.2"
	# docker-compose up -d
else
	# mkdir -p localnet-setup
	 @$(MAKE) localnet-build

#	if ! [ -f localnet-setup/node0/$(UPTICK_BINARY)/config/genesis.json ]; \
#	then \
#		docker run --rm -v $(CURDIR)/localnet-setup:/uptick:Z uptick/node:v0.1 "export LD_LIBRARY_PATH=/wasm && ./uptickd testnet init-files --v 4 -o /uptick --keyring-backend=test --starting-ip-address 192.167.10.2"; \
#	fi
#
#	docker-compose up -d
endif

# Stop testnet
localnet-stop:
	docker-compose down

# Clean testnet
localnet-clean:
	docker-compose down
	sudo rm -rf localnet-setup

 # Reset testnet
localnet-unsafe-reset:
	docker-compose down
ifeq ($(OS),Windows_NT)
	@docker run --rm -v $(CURDIR)\localnet-setup\node0\uptickd:uptick\Z uptick/node:v0.1 "./uptickd unsafe-reset-all --home=/uptick"
	@docker run --rm -v $(CURDIR)\localnet-setup\node1\uptickd:uptick\Z uptick/node:v0.1 "./uptickd unsafe-reset-all --home=/uptick"
	@docker run --rm -v $(CURDIR)\localnet-setup\node2\uptickd:uptick\Z uptick/node:v0.1 "./uptickd unsafe-reset-all --home=/uptick"
	@docker run --rm -v $(CURDIR)\localnet-setup\node3\uptickd:uptick\Z uptick/node:v0.1 "./uptickd unsafe-reset-all --home=/uptick"
else
	@docker run --rm -v $(CURDIR)/localnet-setup/node0/uptickd:/uptick:Z uptick/node:v0.1 "./uptickd unsafe-reset-all --home=/uptick"
	@docker run --rm -v $(CURDIR)/localnet-setup/node1/uptickd:/uptick:Z uptick/node:v0.1 "./uptickd unsafe-reset-all --home=/uptick"
	@docker run --rm -v $(CURDIR)/localnet-setup/node2/uptickd:/uptick:Z uptick/node:v0.1 "./uptickd unsafe-reset-all --home=/uptick"
	@docker run --rm -v $(CURDIR)/localnet-setup/node3/uptickd:/uptick:Z uptick/node:v0.1 "./uptickd unsafe-reset-all --home=/uptick"
endif

# Clean testnet
localnet-show-logstream:
	docker-compose logs --tail=1000 -f

.PHONY: build-docker-local-uptick localnet-start localnet-stop

###############################################################################
###                                Releasing                                ###
###############################################################################

PACKAGE_NAME:=github.com/UptickNetwork/uptick
GOLANG_CROSS_VERSION  = v1.25.8
GOLANG_CROSS_IMAGE    = ghcr.io/goreleaser/goreleaser-cross
GOPATH ?= '$(HOME)/go'
release-dry-run:
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		-e GOMODCACHE=/go/pkg/mod \
		-e GOPROXY="`go env GOPROXY`" \
		-e GOSUMDB="`go env GOSUMDB`" \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-v ${GOPATH}/pkg:/go/pkg \
		-w /go/src/$(PACKAGE_NAME) \
		${GOLANG_CROSS_IMAGE}:${GOLANG_CROSS_VERSION} \
		--clean --snapshot

release:
	@if [ ! -f ".release-env" ]; then \
		echo "\033[91m.release-env is required for release\033[0m";\
		exit 1;\
	fi
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		-e GOMODCACHE=/go/pkg/mod \
		-e GOPROXY="`go env GOPROXY`" \
		-e GOSUMDB="`go env GOSUMDB`" \
		--env-file .release-env \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/$(PACKAGE_NAME) \
		-w /go/src/$(PACKAGE_NAME) \
		${GOLANG_CROSS_IMAGE}:${GOLANG_CROSS_VERSION} \
		release --clean

.PHONY: release-dry-run release

###############################################################################
###                        Compile Solidity Contracts                       ###
###############################################################################

CONTRACTS_DIR := contracts
COMPILED_DIR := contracts/compiled_contracts
TMP := tmp
TMP_CONTRACTS := $(TMP).contracts
TMP_COMPILED := $(TMP)/compiled.json
TMP_JSON := $(TMP)/tmp.json

# Compile and format solidity contracts for the erc20 module. Also install
# openzeppeling as the contracts are build on top of openzeppelin templates.
contracts-compile: contracts-clean openzeppelin create-contracts-json

# Install openzeppelin solidity contracts
openzeppelin:
	@echo "Importing openzeppelin contracts..."
	@cd $(CONTRACTS_DIR)
	@npm install
	@cd ../../../../
	@mv node_modules $(TMP)
	@mv package-lock.json $(TMP)
	@mv $(TMP)/@openzeppelin $(CONTRACTS_DIR)

# Clean tmp files
contracts-clean:
	@rm -rf tmp
	@rm -rf node_modules
	@rm -rf $(COMPILED_DIR)
	@rm -rf $(CONTRACTS_DIR)/@openzeppelin

# Compile, filter out and format contracts into the following format.
# {
# 	"abi": "[{\"inpu 			# JSON string
# 	"bin": "60806040
# 	"contractName": 			# filename without .sol
# }
create-contracts-json:
	@for c in $(shell ls $(CONTRACTS_DIR) | grep '\.sol' | sed 's/.sol//g'); do \
		command -v jq > /dev/null 2>&1 || { echo >&2 "jq not installed."; exit 1; } ;\
		command -v solc > /dev/null 2>&1 || { echo >&2 "solc not installed."; exit 1; } ;\
		mkdir -p $(COMPILED_DIR) ;\
		mkdir -p $(TMP) ;\
		echo "\nCompiling solidity contract $${c}..." ;\
		solc --combined-json abi,bin $(CONTRACTS_DIR)/$${c}.sol > $(TMP_COMPILED) ;\
		echo "Formatting JSON..." ;\
		get_contract=$$(jq '.contracts["$(CONTRACTS_DIR)/'$$c'.sol:'$$c'"]' $(TMP_COMPILED)) ;\
		add_contract_name=$$(echo $$get_contract | jq '. + { "contractName": "'$$c'" }') ;\
		echo $$add_contract_name | jq > $(TMP_JSON) ;\
		abi_string=$$(echo $$add_contract_name | jq -cr '.abi') ;\
		echo $$add_contract_name | jq --arg newval "$$abi_string" '.abi = $$newval' > $(TMP_JSON) ;\
		mv $(TMP_JSON) $(COMPILED_DIR)/$${c}.json ;\
	done
	@rm -rf tmp
