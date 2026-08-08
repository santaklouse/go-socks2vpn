# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.26.5-alpine3.23 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY client ./client
COPY cmd ./cmd
COPY engine ./engine
COPY internal ./internal

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/socks2vpn ./cmd/socks2vpn

FROM alpine:3.23.5

RUN apk add --no-cache ca-certificates curl iproute2 procps-ng

COPY --from=builder /out/socks2vpn /usr/local/bin/socks2vpn

LABEL org.opencontainers.image.source="https://github.com/santaklouse/go-socks2vpn" \
      org.opencontainers.image.description="Route a Linux network namespace through SOCKS4 or SOCKS5"

ENTRYPOINT ["/usr/local/bin/socks2vpn"]
CMD ["--help"]
