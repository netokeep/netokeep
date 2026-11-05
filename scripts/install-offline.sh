#!/usr/bin/env bash
set -euo pipefail

# Env Name
NK_BIN_DIR="${NK_BIN_DIR:-$HOME/.local/bin}"
mkdir -p "$NK_BIN_DIR"

# Platform detection
case "$(uname -s)" in
	Linux) os=linux ;;
	Darwin) os=darwin ;;
	*) printf "\033[31mError:\033[0m Unsupported OS: %s\n" "$(uname -s)"; exit 1 ;;
esac
case "$(uname -m)" in
	x86_64 | amd64) arch=amd64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*) printf "\033[31mError:\033[0m Unsupported architecture: %s\n" "$(uname -m)"; exit 1 ;;
esac
case "$os/$arch" in
	linux/amd64 | darwin/amd64 | darwin/arm64) ;;
	*) printf "\033[31mError:\033[0m Unsupported platform: %s/%s\n" "$os" "$arch"; exit 1 ;;
esac

SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_BINARY="$SRC_DIR/netokeep-server-$os-$arch"
CLIENT_BINARY="$SRC_DIR/netokeep-client-$os-$arch"

if [ ! -f "$SERVER_BINARY" ] && [ ! -f "$CLIENT_BINARY" ]; then
	printf "\033[31mError:\033[0m Neither the server nor the client binary was found in\n\t%s\n" "$SRC_DIR"
	exit 1
fi

# install_bin <src> <name>: copy next to the target first, then rename over it.
# The rename is atomic and safe for a running binary; the user's file is left in place.
install_bin() {
	local tmp
	tmp="$(mktemp "$NK_BIN_DIR/.$2.XXXXXX")"
	printf "\033[1m\033[36m==>\033[0m\033[1m Installing %s...\033[0m\n" "$3"
	if ! cp "$1" "$tmp"; then
		rm -f "$tmp"
		printf "\033[31mError:\033[0m failed to copy %s\n" "$1"
		exit 1
	fi
	chmod 755 "$tmp"
	mv -f "$tmp" "$NK_BIN_DIR/$2"
	INSTALLED+=("$2")
}

INSTALLED=()
if [ -f "$SERVER_BINARY" ]; then
	install_bin "$SERVER_BINARY" nks Server
fi
if [ -f "$CLIENT_BINARY" ]; then
	install_bin "$CLIENT_BINARY" nk Client
fi

# Print installation information (only what was installed in this run)
printf "\033[1m\033[32m\n✔ NetoKeep successfully installed!\033[0m\n\n"
for name in "${INSTALLED[@]}"; do
	printf "    %s: \033[36m%s\033[0m\n" "$name" "$("$NK_BIN_DIR/$name" --version 2>/dev/null | sed -E 's/^.*version[[:space:]]+//')"
done
printf "\n    Location: \033[36m%s\033[0m\n\n" "$NK_BIN_DIR"
printf "Next: Run \033[1m\033[36m%s --help\033[0m to get started.\n" "${INSTALLED[${#INSTALLED[@]}-1]}"

# Detect whether the local bin directory is in the user's PATH
if [[ ":$PATH:" != *":$NK_BIN_DIR:"* ]]; then
	printf "\033[1m\033[33m\nWarning: %s is not in your PATH.\033[0m\n" "$NK_BIN_DIR"
	printf "    You may want to add the following line to your shell configuration file (e.g., ~/.bashrc, ~/.zshrc):\n"
	printf "    \033[36mexport PATH=\"\$PATH:%s\"\033[0m\n" "$NK_BIN_DIR"
fi
