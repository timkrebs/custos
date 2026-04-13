#!/usr/bin/env bash
# Run security scans on custos before release.
# Used in CI and locally before cutting a release tag.

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "=== Security scan for custos release ==="
echo ""

# 1. Vulnerability scan
echo "--- govulncheck ---"
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
echo ""

# 2. Static analysis
echo "--- staticcheck ---"
go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
echo ""

# 3. Go vet
echo "--- go vet ---"
go vet ./...
echo ""

# 4. Dependency verification
echo "--- go mod verify ---"
go mod verify
echo ""

echo "=== All security scans passed ==="
