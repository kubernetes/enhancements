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

VERSION=v0.2.0
URL_BASE=https://raw.githubusercontent.com/kubernetes/repo-infra
URL=$URL_BASE/$VERSION/hack/verify_boilerplate.py
BIN_DIR=bin
SCRIPT=$BIN_DIR/verify_boilerplate.py

if [[ ! -f $SCRIPT ]]; then
    mkdir -p $BIN_DIR
    curl -sfL $URL -o $SCRIPT
    chmod +x $SCRIPT
fi

$SCRIPT --boilerplate-dir hack/boilerplate
