/*
Copyright 2020 The Kubernetes Authors.

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

package prrs

import (
	"strings"
	"testing"
)

func TestValidParsing(t *testing.T) {
	testcases := []struct {
		name         string
		fileContents string
	}{
		{
			"simple test",
			`
kep-number: 12345
beta:
  approver: "@wojtek-t"
stable:
  approver: "johnbelamaric"
`,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			p := &Parser{}
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
