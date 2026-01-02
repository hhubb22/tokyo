#!/bin/bash
#
# Tokyo CLI Installer
# Automatically downloads and installs the latest tokyo binary
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/hhubb22/tokyo/main/install.sh | bash
#   or
#   ./install.sh [version]
#
# Examples:
#   ./install.sh          # Install latest version
#   ./install.sh v0.1.0   # Install specific version
#

set -euo pipefail

# Configuration
REPO="hhubb22/tokyo"
BINARY_NAME="tokyo"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

sha256_of() {
    local file="$1"

    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$file" | cut -d ' ' -f1
        return 0
    fi

    if command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$file" | cut -d ' ' -f1
        return 0
    fi

    return 1
}

# Detect OS
detect_os() {
    local os
    os="$(uname -s)"
    case "$os" in
        Linux*)  echo "Linux" ;;
        Darwin*) echo "Darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "Windows" ;;
        *)
            log_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)  echo "x86_64" ;;
        arm64|aarch64) echo "arm64" ;;
        i386|i686)     echo "i386" ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Get latest version from GitHub API
get_latest_version() {
    local latest
    latest=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ -z "$latest" ]]; then
        log_error "Failed to fetch latest version"
        exit 1
    fi
    echo "$latest"
}

# Download and install
install_tokyo() {
    local version="${1:-}"
    local os
    local arch
    local archive_ext
    local archive_name
    local download_url
    local tmp_dir

    # Detect system
    os=$(detect_os)
    arch=$(detect_arch)

    log_info "Detected OS: $os, Architecture: $arch"

    # Get version
    if [[ -z "$version" ]]; then
        log_info "Fetching latest version..."
        version=$(get_latest_version)
    fi
    log_info "Installing tokyo $version"

    # Determine archive extension
    if [[ "$os" == "Windows" ]]; then
        archive_ext="zip"
    else
        archive_ext="tar.gz"
    fi

    # Build archive name and download URL
    archive_name="${BINARY_NAME}_${os}_${arch}.${archive_ext}"
    download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"

    log_info "Downloading from: $download_url"

    # Create temp directory
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download archive
    if ! curl -fsSL "$download_url" -o "${tmp_dir}/${archive_name}"; then
        log_error "Failed to download ${archive_name}"
        log_error "Please check if the version and platform are correct"
        exit 1
    fi

    local checksums_url
    local checksums_file
    local expected_checksum
    local actual_checksum

    checksums_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"
    checksums_file="${tmp_dir}/checksums.txt"

    if ! curl -fsSL "$checksums_url" -o "$checksums_file"; then
        log_error "Failed to download checksums.txt"
        exit 1
    fi

    expected_checksum=$(grep -E " ${archive_name}$" "$checksums_file" | tr -s ' ' | cut -d ' ' -f1)
    if [[ -z "$expected_checksum" ]]; then
        log_error "Failed to find checksum for ${archive_name}"
        exit 1
    fi

    actual_checksum=$(sha256_of "${tmp_dir}/${archive_name}") || {
        log_error "sha256sum or shasum is required for checksum verification"
        exit 1
    }

    if [[ "$expected_checksum" != "$actual_checksum" ]]; then
        log_error "Checksum verification failed for ${archive_name}"
        exit 1
    fi
    log_info "Checksum verified"

    # Extract archive
    log_info "Extracting archive..."
    cd "$tmp_dir"
    if [[ "$archive_ext" == "zip" ]]; then
        unzip -q "$archive_name"
    else
        tar xzf "$archive_name"
    fi

    # Install binary
    log_info "Installing to ${INSTALL_DIR}..."
    if [[ -w "$INSTALL_DIR" ]]; then
        mv "${BINARY_NAME}" "${INSTALL_DIR}/"
    else
        log_warn "Need sudo permission to install to ${INSTALL_DIR}"
        sudo mv "${BINARY_NAME}" "${INSTALL_DIR}/"
    fi

    # Make executable
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

    # Verify installation
    if command -v tokyo &> /dev/null; then
        log_info "Successfully installed tokyo to ${INSTALL_DIR}/${BINARY_NAME}"
        echo ""
        tokyo --help 2>/dev/null || true
    else
        log_warn "Installation complete, but tokyo is not in PATH"
        log_warn "Add ${INSTALL_DIR} to your PATH or run: ${INSTALL_DIR}/${BINARY_NAME}"
    fi
}

# Uninstall function
uninstall_tokyo() {
    if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
        log_info "Removing ${INSTALL_DIR}/${BINARY_NAME}..."
        if [[ -w "$INSTALL_DIR" ]]; then
            rm -f "${INSTALL_DIR}/${BINARY_NAME}"
        else
            sudo rm -f "${INSTALL_DIR}/${BINARY_NAME}"
        fi
        log_info "tokyo has been uninstalled"
    else
        log_warn "tokyo is not installed at ${INSTALL_DIR}/${BINARY_NAME}"
    fi
}

# Main
main() {
    case "${1:-}" in
        --uninstall|-u)
            uninstall_tokyo
            ;;
        --help|-h)
            echo "Tokyo CLI Installer"
            echo ""
            echo "Usage:"
            echo "  $0 [version]      Install tokyo (latest or specific version)"
            echo "  $0 --uninstall    Uninstall tokyo"
            echo "  $0 --help         Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  INSTALL_DIR       Installation directory (default: /usr/local/bin)"
            echo ""
            echo "Examples:"
            echo "  $0                Install latest version"
            echo "  $0 v0.1.0         Install version v0.1.0"
            echo "  INSTALL_DIR=~/.local/bin $0   Install to custom directory"
            ;;
        *)
            install_tokyo "${1:-}"
            ;;
    esac
}

main "$@"
