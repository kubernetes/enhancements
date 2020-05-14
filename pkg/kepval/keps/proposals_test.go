/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package keps_test

import (
	"strings"
	"testing"

	"k8s.io/enhancements/pkg/kepval/keps"
)

func TestValidParsing(t *testing.T) {
	testcases := []struct {
		name         string
		fileContents string
	}{
		{
			"simple test",
			`---
title: test
authors:
  - "@jpbetz"
  - "@roycaihw"
  - "@sttts"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-api-machinery
  - sig-architecture
reviewers:
  - "@deads2k"
  - "@lavalamp"
  - "@liggitt"
  - "@mbohlool"
  - "@sttts"
approvers:
  - "@deads2k"
  - "@lavalamp"
creation-date: 2018-04-15
last-updated: 2018-04-24
status: provisional

stage: beta
latest-milestone: "v1.19"
milestone:
  alpha: "v1.19"
  beta: "v1.20"
  stable: "v1.22"

feature-gate:
  name: MyFeature
  components:
    - kube-apiserver
    - kube-controller-manager
disable-supported: true
metrics:
  - my_feature_metric
---`,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			p := &keps.Parser{}
			contents := strings.NewReader(tc.fileContents)
			out := p.Parse(contents)
			if out.Error != nil {
				t.Fatalf("expected no error but got one: %v", out.Error)
			}
			if out == nil {
				t.Fatal("out should not be nil")
			}
		})
	}
}
