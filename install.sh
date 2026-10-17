#!/bin/sh
# Install the x-terminal release binary for this machine.
#
#   sh install.sh                    # $HOME/.local/bin
#   sh install.sh --prefix /opt/bin  # somewhere else
#   X_TERMINAL_VERSION=v1.4.0 sh install.sh
#
# What it does: work out the platform, download the matching release asset and its published digest
# from GitHub, verify the digest, and install the binary. It writes one file, it needs no root, and it
# never touches the session file or the configuration.
#
# What the digest proves, and what it does not: the `.sha256` comes from the same release and the same
# host as the binary, so it catches a corrupted or truncated download. It does not prove who published
# it; that is TLS and GitHub's account, exactly as with any release download.
#
# No sudo, no package manager, no container. If you would rather not run a script, the README's
# "Install from a release" section has the same three commands.
set -eu

REPO="LZD121/x-terminal"
PREFIX="${PREFIX:-$HOME/.local/bin}"
VERSION="${X_TERMINAL_VERSION:-}"

usage() {
	cat <<EOF
usage: install.sh [--prefix DIR] [--version TAG]

  --prefix DIR    install into DIR (default: \$HOME/.local/bin)
  --version TAG   install TAG instead of the latest release (e.g. v1.4.0)

environment:
  PREFIX              same as --prefix
  X_TERMINAL_VERSION  same as --version
EOF
}

while [ $# -gt 0 ]; do
	case "$1" in
	--prefix)
		[ $# -ge 2 ] || { echo "install.sh: --prefix needs a directory" >&2; exit 2; }
		PREFIX="$2"
		shift 2
		;;
	--version)
		[ $# -ge 2 ] || { echo "install.sh: --version needs a tag" >&2; exit 2; }
		VERSION="$2"
		shift 2
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "install.sh: unknown argument $1" >&2
		usage >&2
		exit 2
		;;
	esac
done

say() { printf '%s\n' "$*"; }
die() { printf 'install.sh: %s\n' "$*" >&2; exit 1; }

# fetch URL DEST: curl, then wget. A proxy is whatever curl or wget already honours from the
# environment; the script never guesses one.
fetch() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$1" -o "$2"
	elif command -v wget >/dev/null 2>&1; then
		wget -q -O "$2" "$1"
	else
		die "neither curl nor wget is installed"
	fi
}

# digest_check FILE SUMS: verify FILE against the one-line digest file.
digest_check() {
	if command -v sha256sum >/dev/null 2>&1; then
		(cd "$(dirname "$2")" && sha256sum -c "$(basename "$2")")
	elif command -v shasum >/dev/null 2>&1; then
		(cd "$(dirname "$2")" && shasum -a 256 -c "$(basename "$2")")
	else
		die "no sha256 tool (sha256sum or shasum) to verify the download with"
	fi
}

os_name=$(uname -s)
case "$os_name" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) die "unsupported system $os_name" ;;
esac

arch_name=$(uname -m)
case "$arch_name" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) die "unsupported architecture $arch_name" ;;
esac

# The released platforms. A missing pair is a clear error rather than a 404 on a URL nobody checked.
case "$os/$arch" in
linux/amd64 | linux/arm64 | darwin/arm64) ;;
*) die "no release asset for $os/$arch (published: linux/amd64, linux/arm64, darwin/arm64)" ;;
esac

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

if [ -n "$VERSION" ]; then
	case "$VERSION" in
	v*) ;;
	*) VERSION="v$VERSION" ;;
	esac
	base="https://github.com/$REPO/releases/download/$VERSION"
	label="$VERSION"
else
	base="https://github.com/$REPO/releases/latest/download"
	label="the latest release"
fi

# The asset name carries the tag, which `latest` does not name; the SUMS file lists the real names, so
# the script reads it first and takes the line for this platform. That is also what keeps the script
# from having to know the version it is installing.
say "downloading $label for $os/$arch"
fetch "$base/SHA256SUMS" "$tmp/SHA256SUMS"
asset=$(awk -v suffix="_${os}_${arch}" '
	{ for (i = 1; i <= NF; i++) if (index($i, suffix) == length($i) - length(suffix) + 1) { print $i; exit } }
' "$tmp/SHA256SUMS")
[ -n "$asset" ] || die "SHA256SUMS lists no asset for $os/$arch"

fetch "$base/$asset" "$tmp/$asset"
say "verifying $asset"
if fetch "$base/$asset.sha256" "$tmp/$asset.sha256" 2>/dev/null; then
	digest_check "$tmp/$asset" "$tmp/$asset.sha256" || die "digest mismatch for $asset"
	say "verified against $asset.sha256"
else
	# The per-asset digest is missing from a release that predates it; SHA256SUMS carries the same
	# value, from the same release.
	expected=$(awk -v want="$asset" '$NF == want { print $1 }' "$tmp/SHA256SUMS")
	[ -n "$expected" ] || die "no published digest for $asset"
	if command -v sha256sum >/dev/null 2>&1; then
		actual=$(sha256sum "$tmp/$asset" | awk '{ print $1 }')
	else
		actual=$(shasum -a 256 "$tmp/$asset" | awk '{ print $1 }')
	fi
	[ "$expected" = "$actual" ] || die "digest mismatch: published $expected, downloaded $actual"
	say "verified against SHA256SUMS"
fi

mkdir -p "$PREFIX" || die "cannot create $PREFIX"
target="$PREFIX/x-terminal"
say "installing to $target"
# install(1) when it is there, cp+chmod otherwise: both leave a 0755 file, and neither needs root.
if command -v install >/dev/null 2>&1; then
	install -m 0755 "$tmp/$asset" "$target" || die "cannot write $target"
else
	cp "$tmp/$asset" "$target" && chmod 0755 "$target" || die "cannot write $target"
fi

# The binary's own report is the last check: a file that runs and names the release it came from.
if ! version_line=$("$target" --version 2>/dev/null); then
	die "$target was installed but does not run on this machine"
fi
say "$version_line"

case ":$PATH:" in
*":$PREFIX:"*) ;;
*)
	say ""
	say "$PREFIX is not on PATH. Add it:"
	say "  export PATH=\"$PREFIX:\$PATH\""
	;;
esac

say ""
say "next: x-terminal login   (then x-terminal doctor)"
