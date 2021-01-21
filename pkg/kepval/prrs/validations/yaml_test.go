/*
Copyright 2021 The Kubernetes Authors.

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

package validations

import (
	"testing"
)


func TestValidateStructureSuccess(t *testing.T) {
	testcases := []struct {
		name  string
		input map[interface{}]interface{}
	}{
		{
			name: "all milestones optional",
			input: map[interface{}]interface{}{},
		},
		{
			name: "just alpha",
			input: map[interface{}]interface{}{
				"alpha": map[interface{}]interface{}{
					"approver": "@wojtek-t",
				},
			},
		},
		{
			name: "all milestones",
			input: map[interface{}]interface{}{
				"alpha": map[interface{}]interface{}{
					"approver": "@wojtek-t",
				},
				"beta": map[interface{}]interface{}{
					"approver": "@wojtek-t",
				},
				"stable": map[interface{}]interface{}{
					"approver": "@wojtek-t",
				},
			},
		},
		{
			name: "just stable",
			input: map[interface{}]interface{}{
				"stable": map[interface{}]interface{}{
					"approver": "@wojtek-t",
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// Add required fields.
			tc.input["kep-number"] = "12345"

			err := ValidateStructure(tc.input)
			if err != nil {
				t.Fatalf("did not expect an error: %v", err)
			}
		})
	}
}

func TestValidateStructureFailures(t *testing.T) {
	testcases := []struct {
		name  string
		input map[interface{}]interface{}
	}{
		{
			name: "non string key",
			input: map[interface{}]interface{}{
				1: "hello",
			},
		},
		{
			name:  "invalid milestone",
			input: map[interface{}]interface{}{
				"kep-number": "12345",
				"alpha": "@wojtek-t",
			},
		},
		{
			name:  "invalid approver",
			input: map[interface{}]interface{}{
				"kep-number": "12345",
				"alpha": map[interface{}]interface{}{
					"approver": "@xyz",
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStructure(tc.input)
			if err == nil {
				t.Fatal("expecting an error")
			}
		})
	}
}
