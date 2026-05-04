#!/usr/bin/env sh
set -eu

OS=$(uname -s | tr '[:upper:]' '[:lower:]')

if [ "$OS" = "darwin" ]; then
  if ! command -v brew >/dev/null 2>&1; then
    echo "Error: Homebrew required on macOS. Install from https://brew.sh and re-run." >&2
    exit 1
  fi
  exec brew install nqode-io/tap/qode
fi

ARCH=$(uname -m)
case "$ARCH" in
  x86_64)        ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

REPO="nqode-io/qode"
INSTALL_DIR="${HOME}/.local/bin"
API_URL="https://api.github.com/repos/${REPO}/releases?per_page=30"

WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"; rm -f "${INSTALL_DIR}/qode.new"' EXIT

if ! curl -sSfL -H "Accept: application/vnd.github.v3+json" "$API_URL" -o "${WORKDIR}/releases.json"; then
  echo "Error: failed to fetch release list from $API_URL" >&2
  exit 1
fi

# Beta-channel installer: matches any v* tag, including -beta pre-releases.
# Post-beta switch to /releases/latest is documented in docs/versioning.md.
VERSION=$(grep '"tag_name":' "${WORKDIR}/releases.json" \
  | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/' \
  | grep -E '^v[0-9]' \
  | head -n 1)

if [ -z "$VERSION" ]; then
  echo "Error: no tagged v* release found in $REPO" >&2
  exit 1
fi

FILENAME="qode_${VERSION}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/${REPO}/releases/download/${VERSION}"

curl -sSfL "${BASE}/${FILENAME}"    -o "${WORKDIR}/${FILENAME}"
curl -sSfL "${BASE}/checksums.txt"  -o "${WORKDIR}/checksums.txt"

cd "$WORKDIR"
LINE=$(awk -v f="$FILENAME" 'NF==2 && length($1)==64 && $1 ~ /^[0-9a-fA-F]+$/ && $2==f' checksums.txt)
if [ -z "$LINE" ]; then
  echo "Error: checksum entry missing or malformed for $FILENAME" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  printf '%s\n' "$LINE" | sha256sum -c -
elif command -v shasum >/dev/null 2>&1; then
  printf '%s\n' "$LINE" | shasum -a 256 -c -
else
  echo "Error: neither sha256sum nor shasum found" >&2; exit 1
fi

tar -xzf "$FILENAME" qode
chmod +x qode
mkdir -p "$INSTALL_DIR"
mv -f qode "${INSTALL_DIR}/qode.new"
mv -f "${INSTALL_DIR}/qode.new" "${INSTALL_DIR}/qode"

echo "qode ${VERSION} installed to ${INSTALL_DIR}/qode"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "Add to PATH: export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
esac
