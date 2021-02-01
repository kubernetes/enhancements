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

set -euo pipefail

# Default timeout is 1800s
TEST_TIMEOUT=1800

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

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

GO111MODULE=on go test -v -timeout="${TEST_TIMEOUT}s" -count=1 -cover -coverprofile coverage.out "$(go list ./... | grep -v k8s.io/enhancements/cmd)"

go tool cover -html coverage.out -o coverage.html
