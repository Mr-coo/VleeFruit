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
# The microsoft/onnxruntime GitHub releases are no longer reachable, so take
# the same official build from Microsoft's NuGet package (its linux-x64
# libonnxruntime.so is byte-identical to the GitHub tarball's). The checksum
# pins the 1.20.1 package; update it together with ORT_VERSION.
ARG ORT_NUPKG_SHA256=9c32108e9a1c78b090a95c2a219f244dd5f067d4caf68fb0f967103f3f20839c
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl unzip && \
    rm -rf /var/lib/apt/lists/* && \
    curl -fsSL -o /tmp/ort.nupkg \
      "https://api.nuget.org/v3-flatcontainer/microsoft.ml.onnxruntime/${ORT_VERSION}/microsoft.ml.onnxruntime.${ORT_VERSION}.nupkg" && \
    echo "${ORT_NUPKG_SHA256}  /tmp/ort.nupkg" | sha256sum -c - && \
    mkdir -p /ort/lib && \
    unzip -j /tmp/ort.nupkg runtimes/linux-x64/native/libonnxruntime.so -d /ort/lib && \
    cp /ort/lib/libonnxruntime.so "/ort/lib/libonnxruntime.so.${ORT_VERSION}" && \
    cp /ort/lib/libonnxruntime.so /ort/lib/libonnxruntime.so.1 && \
    rm /tmp/ort.nupkg

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

# Create and own the data directory so the SQLite database can be written here by uid 10001
RUN mkdir -p /app/data && chown app:app /app/data

USER app
EXPOSE 8080

ENTRYPOINT ["/app/app"]
