# syntax=docker/dockerfile:1

########################
# Build stage
########################
FROM golang:1.25-alpine AS builder

# Path to the main package to compile.
ARG MAIN_PKG=./cmd/api

WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the rest of the source and build a static binary.
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/app "${MAIN_PKG}"

########################
# Runtime stage
########################
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app

WORKDIR /app
COPY --from=builder /out/app /app/app

USER app
EXPOSE 8080

ENTRYPOINT ["/app/app"]
