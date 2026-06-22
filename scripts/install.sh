#!/usr/bin/env sh
set -eu

repo="${TMC_REPO:-huski-inc/tmcopilot-cli}"
version="${TMC_VERSION:-latest}"
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  darwin) os_name="darwin" ;;
  linux) os_name="linux" ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac

case "$arch" in
  arm64|aarch64) arch_name="arm64" ;;
  x86_64|amd64) arch_name="amd64" ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

if [ "$version" = "latest" ]; then
  version="$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)"
fi

tmp="${TMPDIR:-/tmp}/tmc-install-$$"
mkdir -p "$tmp"
trap 'rm -rf "$tmp"' EXIT

asset_version="${version#v}"
asset="tmc-${asset_version}-${os_name}-${arch_name}.tar.gz"
url="https://github.com/$repo/releases/download/$version/$asset"
checksums_url="https://github.com/$repo/releases/download/$version/checksums.txt"

curl -fsSL "$url" -o "$tmp/$asset"
curl -fsSL "$checksums_url" -o "$tmp/checksums.txt"

if command -v shasum >/dev/null 2>&1; then
  expected="$(awk -v name="$asset" '$2 == name { print $1 }' "$tmp/checksums.txt")"
  actual="$(shasum -a 256 "$tmp/$asset" | awk '{ print $1 }')"
  if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
    echo "checksum verification failed for $asset" >&2
    exit 1
  fi
fi

tar -xzf "$tmp/$asset" -C "$tmp"
chmod +x "$tmp/tmc"

install_dir="${TMC_INSTALL_DIR:-/usr/local/bin}"
mkdir -p "$install_dir"
mv "$tmp/tmc" "$install_dir/tmc"
rm -f "$install_dir/tmcopilot"
if ln -s "$install_dir/tmc" "$install_dir/tmcopilot" 2>/dev/null; then
  :
else
  cp "$install_dir/tmc" "$install_dir/tmcopilot"
  chmod +x "$install_dir/tmcopilot"
fi
echo "installed $version to $install_dir/tmc and $install_dir/tmcopilot"
