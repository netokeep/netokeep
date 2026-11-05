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


# Tools Detection
for dep in curl ; do
	if ! command -v "$dep" &>/dev/null; then
		printf "\033[31mError:\033[0m '%s' is required but not installed.\n" "$dep"
		exit 1
	fi
done


# The server (nks) needs sshd, so it is only released for Linux
COMPONENTS=(nk)
if [ "$os" = linux ]; then
	COMPONENTS=(nks nk)
fi

# version_of <binary>: prints its version, or nothing if it is not installed
version_of() {
	if [ -x "$1" ]; then
		"$1" --version 2>/dev/null | sed -E 's/^.*version[[:space:]]+//' || true
	fi
}

# Latest release version, from where /releases/latest redirects to (.../tag/v1.2.6).
# Empty if GitHub cannot be reached; installation then continues anyway.
latest_url="$(curl -fsIL -o /dev/null -w "%{url_effective}" "https://github.com/netokeep/netokeep/releases/latest" || true)"
LATEST_VERSION="${latest_url##*/v}"
if [ "$LATEST_VERSION" = "$latest_url" ]; then
	LATEST_VERSION=""
fi
if [ -z "$LATEST_VERSION" ]; then
	printf "\033[33mCould not determine the latest version; continuing with the download.\033[0m\n"
fi


# install_bin <name> <asset> <label>: download next to the target, then rename over it (atomic)
install_bin() {
	local tmp url="https://github.com/netokeep/netokeep/releases/latest/download/$2"
	tmp="$(mktemp "$NK_BIN_DIR/.$1.XXXXXX")"
	printf "\033[1m\033[36m==>\033[0m\033[1m Downloading %s...\033[0m\n" "$3"
	if ! curl -fL --progress-bar -o "$tmp" "$url"; then
		rm -f "$tmp"
		printf "\033[31mError:\033[0m download failed: %s\n" "$url"
		exit 1
	fi
	chmod 755 "$tmp"
	mv -f "$tmp" "$NK_BIN_DIR/$1"
}

INSTALLED=()
for name in "${COMPONENTS[@]}"; do
	current="$(version_of "$NK_BIN_DIR/$name")"
	if [ -n "$LATEST_VERSION" ] && [ "$current" = "$LATEST_VERSION" ]; then
		printf "\033[1m\033[32m✔ %s is already up-to-date (version %s).\033[0m\n" "$name" "$current"
		continue
	fi
	if [ -n "$current" ] && [ -n "$LATEST_VERSION" ]; then
		printf "\033[36mUpdating %s: %s -> %s\033[0m\n" "$name" "$current" "$LATEST_VERSION"
	fi
	if [ "$name" = nks ]; then
		install_bin nks "netokeep-server-$os-$arch" Server
	else
		install_bin nk "netokeep-client-$os-$arch" Client
	fi
	INSTALLED+=("$name")
done
if [ ${#INSTALLED[@]} -eq 0 ]; then
	exit 0
fi


# Print installation information (only what was installed in this run)
printf "\033[1m\033[32m\n✔ NetoKeep successfully installed!\033[0m\n\n"
for name in "${INSTALLED[@]}"; do
	printf "    %s: \033[36m%s\033[0m\n" "$name" "$(version_of "$NK_BIN_DIR/$name")"
done
printf "\n    Location: \033[36m%s\033[0m\n\n" "$NK_BIN_DIR"
printf "Next: Run \033[1m\033[36m%s --help\033[0m to get started.\n" "${INSTALLED[${#INSTALLED[@]}-1]}"

# Detect whether the local bin directory is in the user's PATH
if [[ ":$PATH:" != *":$NK_BIN_DIR:"* ]]; then
	printf "\033[1m\033[33m\nWarning: %s is not in your PATH.\033[0m\n" "$NK_BIN_DIR"
	printf "    You may want to add the following line to your shell configuration file (e.g., ~/.bashrc, ~/.zshrc):\n"
	printf "    \033[36mexport PATH=\"\$PATH:%s\"\033[0m\n" "$NK_BIN_DIR"
fi
