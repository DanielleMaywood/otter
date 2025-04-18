#!/usr/bin/env bash

set -euo pipefail

VERSION=$(./scripts/version.sh)

LDFLAGS=()
LDFLAGS+=(-X "'github.com/DanielleMaywood/otter/internal/buildinfo.version=$VERSION'")

BUILDARGS=()
BUILDARGS+=(-ldflags "${LDFLAGS[*]}")

go build "${BUILDARGS[@]}" ./cmd/otter
