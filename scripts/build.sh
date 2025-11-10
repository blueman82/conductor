#!/usr/bin/env bash
#
# build.sh - Cross-compilation build script for Conductor
#
# Usage:
#   ./scripts/build.sh              # Build for all platforms
#   ./scripts/build.sh linux amd64  # Build for specific platform
#

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Binary name
BINARY_NAME="conductor"

# Output directory
DIST_DIR="${PROJECT_ROOT}/dist"

# Get version from git tag or default
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")}"

# Build time
BUILD_TIME="$(date -u '+%Y-%m-%d_%H:%M:%S')"

# Git commit
GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")"

# LDFLAGS for version injection
LDFLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# Platform matrix: OS/ARCH pairs
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print with color
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Build for a specific platform
build_platform() {
    local os=$1
    local arch=$2
    local output_name="${BINARY_NAME}-${VERSION}-${os}-${arch}"

    # Add .exe extension for Windows
    if [ "$os" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    local output_path="${DIST_DIR}/${output_name}"

    print_info "Building for ${os}/${arch}..."

    # Set environment variables and build
    GOOS=$os GOARCH=$arch go build \
        -ldflags "${LDFLAGS}" \
        -o "${output_path}" \
        "${PROJECT_ROOT}/cmd/conductor"

    if [ $? -eq 0 ]; then
        local size=$(du -h "${output_path}" | cut -f1)
        print_info "Built ${output_name} (${size})"
    else
        print_error "Failed to build for ${os}/${arch}"
        return 1
    fi
}

# Main function
main() {
    print_info "Conductor Cross-Compilation Build Script"
    print_info "Version: ${VERSION}"
    print_info "Build Time: ${BUILD_TIME}"
    print_info "Git Commit: ${GIT_COMMIT}"
    echo ""

    # Create dist directory
    mkdir -p "${DIST_DIR}"

    # Check if specific platform requested
    if [ $# -eq 2 ]; then
        local os=$1
        local arch=$2
        print_info "Building for specific platform: ${os}/${arch}"
        build_platform "$os" "$arch"
    else
        # Build for all platforms
        print_info "Building for all platforms..."
        echo ""

        local success_count=0
        local fail_count=0

        for platform in "${PLATFORMS[@]}"; do
            local os="${platform%/*}"
            local arch="${platform#*/}"

            if build_platform "$os" "$arch"; then
                ((success_count++))
            else
                ((fail_count++))
            fi
            echo ""
        done

        # Summary
        print_info "Build Summary:"
        print_info "  Success: ${success_count}"
        if [ $fail_count -gt 0 ]; then
            print_error "  Failed: ${fail_count}"
        fi
    fi

    # List all built binaries
    echo ""
    print_info "Built binaries in ${DIST_DIR}:"
    ls -lh "${DIST_DIR}" | tail -n +2 || true

    echo ""
    print_info "Build complete!"
}

# Run main function
main "$@"
