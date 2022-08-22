# Copyright 2022 The Kubernetes Authors.
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


IMAGES += components.png kubelet.png

all: $(IMAGES)
clean:
	rm -f $(IMAGES)

# We use the http://plantuml.com/plantuml server to generate
# images. That way nothing needs to be installed besides Go.
DOC_PLANTUML_GO = $(shell go env GOPATH)/bin/plantuml-go

%.png: %.puml $(DOC_PLANTUML_GO)
	$(DOC_PLANTUML_GO) -format png $<

%.svg: %.puml $(DOC_PLANTUML_GO)
	$(DOC_PLANTUML_GO) -format svg $<

# Builds the binary in GOPATH/bin. Changing into / first avoids
# modifying the project's go.mod file.
$(DOC_PLANTUML_GO):
	cd / && go get github.com/acarlson99/plantuml-go
