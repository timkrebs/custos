# Dockerfile for custos — used by GoReleaser.
# GoReleaser copies the pre-built binary into the build context,
# so this image just packages it into a minimal Alpine container.
#
# Manual build (outside GoReleaser):
#   make build
#   docker build -f .release/docker/Dockerfile -t custos --build-arg BINARY=bin/custos .
#
# Run:
#   docker run --rm -v $(pwd):/work ghcr.io/timkrebs/custos test -f /work/spec.yaml

FROM alpine:3.21

# Non-root user for security
RUN addgroup -S custos && adduser -S custos -G custos

# GoReleaser places the binary as "custos" in the build context
ARG BINARY=custos
COPY ${BINARY} /usr/local/bin/custos

USER custos
WORKDIR /work

ENTRYPOINT ["custos"]
CMD ["--help"]
