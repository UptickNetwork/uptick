FROM golang:1.24-bookworm AS build-env

WORKDIR /go/src/github.com/UptickNetwork/uptick

RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

COPY . .

RUN make build

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates jq && rm -rf /var/lib/apt/lists/*

WORKDIR /root

COPY --from=build-env /go/src/github.com/UptickNetwork/uptick/build/uptickd /usr/bin/uptickd

EXPOSE 26656 26657 1317 9090

CMD ["uptickd"]