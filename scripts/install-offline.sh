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

# ensure_not_running <name>: refuse to install while any "<name> core" process is running,
# wherever its binary lives (users may move it); stop it first instead of leaving it orphaned.
ensure_not_running() {
	local pids="" d args a0 a1
	if [ "$os" = linux ]; then
		# Read /proc directly: minimal containers often have no pgrep
		for d in /proc/[0-9]*; do
			args="$(tr '\0' '\n' 2>/dev/null < "$d/cmdline" | head -n 2 || true)"
			a0="${args%%$'\n'*}"
			a1="${args#*$'\n'}"
			if [ "${a0##*/}" = "$1" ] && [ "$a1" = core ]; then
				pids="$pids ${d#/proc/}"
			fi
		done
	else
		# argv[0] ends with /<name> (or is <name>) and argv[1] is "core"
		pids="$(pgrep -f -- "(^|/)$1 core( |\$)" | tr '\n' ' ' || true)"
	fi
	pids="$(echo $pids)" # trim spaces
	if [ -n "$pids" ]; then
		local hint="nks stop"
		if [ "$1" = nk ]; then hint="nk stop <name>"; fi
		printf "\033[31mError:\033[0m %s is running (PID: %s). Stop it first with \033[36m%s\033[0m, then rerun the installer.\n" "$1" "$pids" "$hint"
		exit 1
	fi
}

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
	# nks runs /usr/sbin/sshd (that exact path) for SSH sessions
	if [ ! -x /usr/sbin/sshd ]; then
		printf "\033[33mWarning: /usr/sbin/sshd not found; nks needs it to serve SSH.\033[0m\n"
		printf "    Install it first, e.g. \033[36mapt install openssh-server\033[0m\n"
	fi
	ensure_not_running nks
	install_bin "$SERVER_BINARY" nks Server
fi
if [ -f "$CLIENT_BINARY" ]; then
	ensure_not_running nk
	install_bin "$CLIENT_BINARY" nk Client
fi

# Print installation information (only what was installed in this run)
printf "\033[1m\033[32m\n✔ NetoKeep successfully installed!\033[0m\n\n"
for name in "${INSTALLED[@]}"; do
	if [ "$name" = nks ]; then label="Server version"; else label="Client version"; fi
	printf "    %s: \033[36m%s\033[0m\n" "$label" "$("$NK_BIN_DIR/$name" --version 2>/dev/null | sed -E 's/^.*version[[:space:]]+//')"
done
printf "    Location: \033[36m%s\033[0m\n\n" "$NK_BIN_DIR"
# e.g. "nks --help or nk --help" when both were installed
help=""
for name in "${INSTALLED[@]}"; do
	help+="${help:+ or }\033[1m\033[36m$name --help\033[0m"
done
printf "Next: Run %b to get started.\n" "$help"

# Detect whether the local bin directory is in the user's PATH
if [[ ":$PATH:" != *":$NK_BIN_DIR:"* ]]; then
	printf "\033[1m\033[33m\nWarning: %s is not in your PATH.\033[0m\n" "$NK_BIN_DIR"
	printf "    You may want to add the following line to your shell configuration file (e.g., ~/.bashrc, ~/.zshrc):\n"
	printf "    \033[36mexport PATH=\"\$PATH:%s\"\033[0m\n" "$NK_BIN_DIR"
fi
