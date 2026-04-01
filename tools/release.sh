#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: tools/release.sh <version>"
  echo "  Example: tools/release.sh v0.1.4-alpha"
  exit 1
}

# Validate argument.
if [[ $# -ne 1 ]]; then
  usage
fi

VERSION="$1"

# Validate version format: v<major>.<minor>.<patch> with optional pre-release suffix.
if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
  echo "Error: invalid version format '${VERSION}'"
  echo "       Expected: v<major>.<minor>.<patch>[-prerelease]"
  echo "       Examples: v1.0.0  v0.2.0-beta  v0.1.4-alpha"
  exit 1
fi

# Require main branch.
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
  echo "Error: releases must be tagged from main (currently on '${CURRENT_BRANCH}')"
  exit 1
fi

# Require clean working tree.
if [[ -n $(git status --porcelain) ]]; then
  echo "Error: working tree is dirty — commit or stash changes before releasing"
  git status --short
  exit 1
fi

# Check tag does not already exist.
if git tag -l "$VERSION" | grep -q "^${VERSION}$"; then
  echo "Error: tag '${VERSION}' already exists"
  exit 1
fi

# Show the commit that will be tagged.
echo "Tagging commit:"
git log -1 --oneline
echo ""

# Create and push the tag.
git tag "$VERSION"
git push origin "$VERSION"

echo ""
echo "Released: ${VERSION}"
