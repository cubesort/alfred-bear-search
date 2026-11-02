#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
root_dir="$PWD"

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "Run this script on the Mac where you use Alfred."
  exit 1
fi
if ! command -v go >/dev/null 2>&1; then
  echo "Install Go 1.21 or newer from https://go.dev/dl/, then run this script again."
  exit 1
fi
if ! /usr/bin/xcrun --sdk macosx --find clang >/dev/null 2>&1; then
  echo "Install Apple's Command Line Tools with: xcode-select --install"
  echo "After installation finishes, run this script again."
  exit 1
fi

if ! sdk_path="$(/usr/bin/xcrun --sdk macosx --show-sdk-path)"; then
  echo "Unable to locate the macOS SDK in the active developer tools."
  echo "Check the active tools with: xcode-select -p"
  echo "Install or update Apple's Command Line Tools, then try again."
  exit 1
fi
for header in errno.h stdlib.h pthread.h; do
  if [[ ! -f "$sdk_path/usr/include/$header" ]]; then
    echo "The selected macOS SDK is missing $header: $sdk_path"
    echo "Check the active tools with: xcode-select -p"
    echo "Install or update Apple's Command Line Tools (or select a complete Xcode installation)."
    exit 1
  fi
done

# Finding clang alone does not ensure that cgo can find its system headers.
# Select the active macOS SDK explicitly, including for the linker. Quote the
# SDK path inside cgo's flag string to support Xcode locations containing spaces.
export SDKROOT="$sdk_path"
export CGO_CFLAGS="${CGO_CFLAGS:--O2 -g} -isysroot \"$SDKROOT\""
export CGO_LDFLAGS="${CGO_LDFLAGS:--O2 -g} -isysroot \"$SDKROOT\""
export CGO_ENABLED=1
export GOOS=darwin
export GOARCH="$(go env GOHOSTARCH)"
export GOTOOLCHAIN=local
compiler_path="$(/usr/bin/xcrun --sdk macosx --find clang)"
export CC="\"$compiler_path\""

echo "Using macOS SDK: $SDKROOT"
if ! "$compiler_path" -isysroot "$SDKROOT" -x c -fsyntax-only - <<'C_HEADERS'
#include <errno.h>
#include <stdlib.h>
#include <pthread.h>
int main(void) { return 0; }
C_HEADERS
then
  echo "The selected compiler cannot compile against the macOS SDK."
  echo "Check xcode-select -p and update or repair the selected Apple developer tools."
  exit 1
fi

stage_dir="$(mktemp -d "$root_dir/.build.XXXXXX")"
trap 'rm -rf "$stage_dir"' EXIT

echo "Checking the search implementation..."
go test -mod=vendor ./db ./core ./cmd/search -count=1

cp -R workflow-template/. "$stage_dir/"
mkdir -p "$stage_dir/cmd/search"
echo "Building for macOS $GOARCH..."
go build -mod=vendor -buildvcs=false -trimpath -ldflags='-s -w' -o "$stage_dir/cmd/search/search" ./cmd/search
chmod 755 "$stage_dir/cmd/search/search"
/usr/bin/file "$stage_dir/cmd/search/search"
/usr/bin/plutil -lint "$stage_dir/info.plist"

mkdir -p "$root_dir/dist"
output_path="$root_dir/dist/Bear.alfredworkflow"
archive_path="$stage_dir/Bear.alfredworkflow"
(
  cd "$stage_dir"
  /usr/bin/zip -q -r -X "$archive_path" LICENSE icon.png info.plist cmd
)
/usr/bin/unzip -tq "$archive_path"
mv "$archive_path" "$output_path"
echo
echo "Build complete: $output_path"
echo "Double-click the workflow file to import Bear into Alfred."
echo "Search: b <query>"
echo "Return: open in Bear's main window"
echo "Command-Return: open in a new window"
echo "Option-Return: search in Bear"
