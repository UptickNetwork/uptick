FROM golang:1.26.9-bookworm@sha256:d9c68c2c51161e12fd77e4c6320687c9cd86e1af1e3ad6e6cd63ff970641453c AS build-env

WORKDIR /go/src/github.com/UptickNetwork/uptick

RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

COPY . .

RUN make build

FROM debian:bookworm-slim@sha256:3783cc01769c7b2b1b83a5c5ad96c815348e28ed7da68e2e3687004faa906251

RUN apt-get update && apt-get install -y ca-certificates jq && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 uptick

COPY --from=build-env /go/src/github.com/UptickNetwork/uptick/build/uptickd /usr/bin/uptickd

USER uptick

WORKDIR /home/uptick

EXPOSE 26656 26657 1317 9090

HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
  CMD uptickd status --node tcp://127.0.0.1:26657 >/dev/null 2>&1 || exit 1

CMD ["uptickd"]
