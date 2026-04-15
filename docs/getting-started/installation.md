---
description: Install custos on macOS, Linux, Windows, Docker, or from source.
icon: download
---

# Installation

custos ships as a single static binary with no runtime dependencies. Pick whichever method suits your environment.

## Install script (Linux and macOS)

The fastest way. Downloads the latest release for your OS and architecture and drops the binary in `/usr/local/bin` (or `$HOME/.local/bin` without sudo).

```bash
curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash
```

Pin a specific version:

```bash
curl -sSfL https://raw.githubusercontent.com/timkrebs/custos/main/.build/install.sh | bash -s -- v0.1.0
```

{% hint style="warning" %}
Piping a script from the internet into a shell is convenient but you should always read what you are running first. The script is short and auditable at the URL above.
{% endhint %}

## Homebrew (macOS and Linux)

```bash
brew install timkrebs/tap/custos
```

Upgrades follow the normal Homebrew flow:

```bash
brew upgrade custos
```

## Docker

Pull the image from the GitHub Container Registry and mount your policy directory:

```bash
docker run --rm -v "$(pwd):/work" \
  ghcr.io/timkrebs/custos:latest \
  test -f /work/spec.yaml
```

The `:latest` tag tracks the most recent stable release. Pin to a version for reproducible CI:

```bash
docker run --rm -v "$(pwd):/work" \
  ghcr.io/timkrebs/custos:v0.1.0 \
  test -f /work/spec.yaml
```

## From source

If you have Go 1.25 or newer installed:

```bash
go install github.com/timkrebs/custos@latest
```

The binary lands in `$(go env GOBIN)` (or `$HOME/go/bin` by default). Make sure that directory is on your `PATH`.

To build a specific branch or tag from a clone:

```bash
git clone https://github.com/timkrebs/custos.git
cd custos
make build
./bin/custos version
```

## Windows

Download the Windows zip from the [releases page](https://github.com/timkrebs/custos/releases) and extract `custos.exe` to a directory on your `PATH`. PowerShell example:

```powershell
Invoke-WebRequest -Uri https://github.com/timkrebs/custos/releases/latest/download/custos_windows_amd64.zip -OutFile custos.zip
Expand-Archive custos.zip -DestinationPath $env:USERPROFILE\bin
```

## Verify the install

```bash
custos version
```

You should see the version string, commit hash, and build timestamp. If custos is not found, make sure the install directory is on your `PATH` and open a new shell.

## Upgrading

- **Install script:** re-run it. The script replaces the existing binary in place.
- **Homebrew:** `brew upgrade custos`.
- **Docker:** pull the newer tag.
- **From source:** `go install github.com/timkrebs/custos@latest`.

## Uninstalling

Remove the binary from wherever it was installed:

```bash
rm "$(command -v custos)"
```

Homebrew users run `brew uninstall custos`. Docker users delete the image with `docker rmi ghcr.io/timkrebs/custos`.
