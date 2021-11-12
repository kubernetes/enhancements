#!/bin/bash

# Check that the required section headers (from the kep template) are
# present in the kep.

set -o errexit
set -o nounset
set -o pipefail

TEMPLATE=/Users/arsh/Projects/Temp/kep-bash/kep-template.md

KEP=/Users/arsh/Projects/Temp/kep-bash/keps/kep.md

RC=0

echo "Checking ${KEP}"

# Check for title which starts with #
if ! grep -q '^# ' $KEP; then
    echo "$KEP is missing a title"
    RC=1
fi

# All other non optional headings start with ##, ### or ######
missing=$(grep -E '^##(#|####)? ' $TEMPLATE |
    grep -v 'Optional' |
    while read header_line; do
        if ! grep -q "^${header_line}" $KEP; then
            echo "$KEP missing \"$header_line\""
        fi
    done)
if [ -n "$missing" ]; then
    echo "$missing"
    RC=1
fi

exit $RC
