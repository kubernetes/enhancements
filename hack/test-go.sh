#!/usr/bin/env bash

# Copyright 2021 The Kubernetes Authors.
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

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)

# Default timeout is 1800s
TEST_TIMEOUT=${TIMEOUT:-1800}

# Write go test artifacts here
ARTIFACTS=${ARTIFACTS:-"${REPO_ROOT}/tmp"}

for arg in "$@"
do
    case $arg in
        -t=*|--timeout=*)
        TEST_TIMEOUT="${arg#*=}"
        shift
        ;;
        -t|--timeout)
        TEST_TIMEOUT="$2"
        shift
        shift
    esac
done

cd "${REPO_ROOT}"

mkdir -p "${ARTIFACTS}"

go_test_flags=(
    -v
    -count=1
    -timeout="${TEST_TIMEOUT}s"
    -cover -coverprofile "${ARTIFACTS}/coverage.out"
)

packages=()
mapfile -t packages < <(go list ./... | grep -v k8s.io/enhancements/cmd)

GO111MODULE=on go test "${go_test_flags[@]}" "${packages[@]}"

go tool cover -html "${ARTIFACTS}/coverage.out" -o "${ARTIFACTS}/coverage.html"
