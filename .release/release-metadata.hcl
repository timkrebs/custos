# Release metadata for custos.
# This file documents the release configuration and is referenced by CI/CD.

project = "custos"

url     = "https://github.com/timkrebs/custos"
license = "MPL-2.0"

docs = "https://timkrebs.github.io/custos"

# Supported platforms — must match .goreleaser.yml
platforms = [
  "linux/amd64",
  "linux/arm64",
  "darwin/amd64",
  "darwin/arm64",
  "windows/amd64",
  "windows/arm64",
]

# Distribution channels
distribution {
  github_releases = true
  homebrew_tap    = "timkrebs/homebrew-tap"
  docker_image    = "ghcr.io/timkrebs/custos"
}
