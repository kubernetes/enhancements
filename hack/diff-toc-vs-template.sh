#!/usr/bin/env bash

# Copyright 2025 The Kubernetes Authors.
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

# keep in sync with hack/verify-toc.sh
# TODO: dedupe
TOOL_VERSION=v1.1.0

# cd to the root path
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
cd "${ROOT}"

# create a temporary directory
TMP_DIR=$(mktemp -d)
# cleanup
exitHandler() (
  echo "Cleaning up..."
  rm -rf "${TMP_DIR}"
)
trap exitHandler EXIT
# Perform go install in a temp dir as we are not tracking this version in a go
# module.
# If we do the go install in the repo, it will create/update go.mod and go.sum.
cd "${TMP_DIR}"
GO111MODULE=on GOBIN="${TMP_DIR}" go install "sigs.k8s.io/mdtoc@${TOOL_VERSION}"
export PATH="${TMP_DIR}:${PATH}"
cd "${ROOT}"

# Identify KEP files changed by the PR:
# default from prow env if unset from args
# https://docs.prow.k8s.io/docs/jobs/#job-environment-variables
# TODO: handle batch PR testing
target=HEAD
base="${BASE_COMMIT:-}"
if [[ -z "${target:-}" && -n "${PULL_PULL_SHA:-}" ]]; then
    target="${PULL_PULL_SHA}"
fi
# target must be a something that git can resolve to a commit.
# "git rev-parse --verify" checks that and prints a detailed
# error.
if [[ -n "${target}" ]]; then
    target="$(git rev-parse --verify "${target}")"
fi
if [[ -z "${base}" && -n "${PULL_BASE_SHA:-}" && -n "${PULL_PULL_SHA:-}" ]]; then
    if ! base="$(git merge-base "${PULL_BASE_SHA}" "${PULL_PULL_SHA}")"; then
        echo >&2 "Failed to detect base revision correctly with prow environment variables."
        exit 1
    fi
elif [[ -z "${base}" ]]; then
    # origin is the default remote, but we encourage our contributors
    # to have both origin (their fork) and upstream, if upstream is present
    # then prefer upstream
    # if they have called it something else, there's no good way to be sure ...
    remote='origin'
    if git remote | grep -q 'upstream'; then
        remote='upstream'
    fi
    default_branch="$(git rev-parse --abbrev-ref "${remote}"/HEAD | cut -d/ -f2)"
    if ! base="$(git merge-base "${remote}/${default_branch}" "${target:-HEAD}")"; then
        echo >&2 "Could not determine default base revision. -r must be used explicitly."
        exit 1
    fi
fi
base="$(git rev-parse --verify "${base}")"

echo "base: $base  target: $target"

# get TOC for template
template_toc=$(mdtoc 'keps/NNNN-kep-template/README.md')

result=0
# get KEP README files changed in the diff
kep_readmes=()
while IFS= read -r changed_file
do
    # make sure to ignore the template kep itself, we don't want to self-diff
    if [[ "$changed_file" == "keps"*"README.md" ]] && [[ "$changed_file" != "keps/NNNN-kep-template/README.md" ]]; then
        kep_readmes+=("$changed_file")
    fi
done < <(git diff-tree --no-commit-id --name-only -r "${base}".."${target}")

for kep_readme in "${kep_readmes[@]}"; do
    kep_toc=$(mdtoc "${kep_readme}")
    echo >&2 "Diffing table of contents for $kep_readme:"
    # diff only removals versus the template
    # we don't care about _additional_ headings in the KEP
    # we also don't care if (Optional) headings are missing
    git diff <(echo "${template_toc}" ) <(echo "${kep_toc}" ) \
        | grep -E '^-' \
        | grep -v '(Optional)' \
      || result=-1
done


echo >&2 "Checked: ${kep_readmes[@]}"
echo >&2 "Result: ${result}"
exit "${result}"

