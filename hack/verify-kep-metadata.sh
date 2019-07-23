#!/usr/bin/env bash

# Copyright 2019 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

TOOL_VERSION=b97ce7d3f1daca6bc79d60952e461df3ffa88a5b

# cd to the root path
ROOT=$(dirname "${BASH_SOURCE}")/..
cd ${ROOT}

GO111MODULE=on go get "github.com/chuckha/kepview/cmd/kepval@${TOOL_VERSION}"

echo "Checking metadata validity..."
# * ignore "0023-documentation-for-images.md" because it is not a real KEP
# * ignore "YYYYMMDD-kep-template.md" because it is not a real KEP
grep --recursive --files-with-matches --regexp '---' --include='*.md' keps | grep --invert-match "YYYYMMDD-kep-template.md" | grep --invert-match "0023-documentation-for-images.md" | xargs kepval