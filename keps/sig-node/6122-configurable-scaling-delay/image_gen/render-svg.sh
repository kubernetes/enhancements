#!/usr/bin/env bash
# Copyright The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Render an SVG to a PNG, using whichever renderer is installed.
#
# Usage: render-svg.sh <in.svg> <out.png> [scale]

set -euo pipefail

svg=$1
png=$2
scale=${3:-2}

attr() {
  grep -o "<svg[^>]*$1=\"[0-9.]*\"" "$svg" | grep -o "$1=\"[0-9.]*\"" |
    grep -o '[0-9.]*' | head -1
}

w=$(attr width)
h=$(attr height)
[ -n "$w" ] && [ -n "$h" ] || { echo "cannot read size from $svg" >&2; exit 1; }
pw=$(awk "BEGIN{printf \"%d\", $w * $scale}")
ph=$(awk "BEGIN{printf \"%d\", $h * $scale}")

if command -v rsvg-convert >/dev/null; then
  rsvg-convert -w "$pw" -h "$ph" -b white -o "$png" "$svg"
  tool=rsvg-convert
elif command -v inkscape >/dev/null; then
  inkscape --export-type=png --export-background=white \
    --export-width="$pw" --export-filename="$png" "$svg"
  tool=inkscape
else
  for c in chromium chromium-browser google-chrome chrome; do
    command -v "$c" >/dev/null && { tool=$c; break; }
  done
  [ -n "${tool:-}" ] || {
    echo "need one of: rsvg-convert, inkscape, chromium" >&2
    exit 1
  }
  # A snap-packaged browser has a private /tmp and cannot write outside
  # $HOME, so render in place, next to the sources.
  dir=$(cd "$(dirname "$png")" && pwd)
  profile=$(mktemp -d "$dir/.render-profile.XXXXXX")
  trap 'rm -rf "$profile"' EXIT
  "$tool" --headless --disable-gpu --no-sandbox --hide-scrollbars \
    --user-data-dir="$profile" --default-background-color=FFFFFFFF \
    --force-device-scale-factor="$scale" --window-size="$w,$h" \
    --screenshot="$dir/$(basename "$png")" \
    "file://$(cd "$(dirname "$svg")" && pwd)/$(basename "$svg")" >/dev/null 2>&1
fi

echo "rendered $png (${pw}x${ph}) with $tool"
