# syntax=docker/dockerfile:1
#
# Builds the backend with in-process ONNX inference (CGO on) and bundles the
# ONNX Runtime shared library. Mount a model and set MODEL_PATH at runtime.

ARG ORT_VERSION=1.20.1
ARG MAIN_PKG=./cmd/server

########################
# Build stage (glibc + gcc, CGO on)
########################
FROM golang:1.26-bookworm AS builder

ARG MAIN_PKG

WORKDIR /src

# gcc/build tools for cgo (onnxruntime_go).
RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev && \
    rm -rf /var/lib/apt/lists/*

# Cache dependencies first.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source and build.
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -tags onnx -ldflags="-s -w" -o /out/app "${MAIN_PKG}"

########################
# ONNX Runtime shared library
########################
FROM debian:bookworm-slim AS ort
ARG ORT_VERSION
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/* && \
    curl -fsSL -o /tmp/ort.tgz \
      "https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/onnxruntime-linux-x64-${ORT_VERSION}.tgz" && \
    mkdir -p /ort && tar -xzf /tmp/ort.tgz -C /ort --strip-components=1 && \
    rm /tmp/ort.tgz

########################
# Runtime stage (glibc)
########################
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/* && \
    useradd -u 10001 -m app

WORKDIR /app
COPY --from=builder /out/app /app/app
COPY --from=ort /ort/lib/libonnxruntime.so* /app/lib/

ENV ONNXRUNTIME_LIB_PATH=/app/lib/libonnxruntime.so

USER app
EXPOSE 8080

ENTRYPOINT ["/app/app"]
