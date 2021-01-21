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

REPO_ROOT	:= $(shell git rev-parse --show-toplevel)

.DEFAULT_GOAL	:= help

.PHONY: targets
targets: help verify verify-toc verify-spelling verify-metadata update-toc add-verify-hook

help: ## Show this help text.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

verify: ## Runs all verification tests.
	${REPO_ROOT}/hack/verify.sh

verify-toc: ## Verifies the Table of Contents is in the correct format.
	${REPO_ROOT}/hack/verify-toc.sh

verify-spelling: ## Verifies spelling.
	${REPO_ROOT}/hack/verify-spelling.sh

verify-metadata: ## Verifies the KEP metadata is valid yaml.
	${REPO_ROOT}/hack/verify-kep-metadata.sh

update-toc: ## Updates KEP Table of Contents.
	${REPO_ROOT}/hack/update-toc.sh

add-verify-hook: ## Adds verify scripts to git pre-commit hooks.
# Note: The pre-commit hooks can be bypassed by using the flag --no-verify when
# performing a git commit.
	git config --local core.hooksPath "${REPO_ROOT}/.githooks"
