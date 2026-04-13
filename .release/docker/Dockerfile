# Multi-stage build for custos Docker image.
# Produces a minimal scratch-based image with only the custos binary.
#
# Build:
#   docker build -f .release/docker/Dockerfile -t custos .
#
# Run:
#   docker run --rm -v $(pwd):/work custos test -f /work/spec.yaml

# --- Build stage ---
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w \
      -X github.com/timkrebs/custos/version.Version=${VERSION} \
      -X github.com/timkrebs/custos/version.GitCommit=${GIT_COMMIT} \
      -X github.com/timkrebs/custos/version.GitTreeState=clean \
      -X github.com/timkrebs/custos/version.BuildDate=${BUILD_DATE}" \
    -o /custos .

# --- Runtime stage ---
FROM alpine:3.21

# Non-root user for security
RUN addgroup -S custos && adduser -S custos -G custos

COPY --from=builder /custos /usr/local/bin/custos

USER custos
WORKDIR /work

ENTRYPOINT ["custos"]
CMD ["--help"]
