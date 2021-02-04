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

package validations

import (
	"bytes"
	"html/template"
	"testing"

	"gopkg.in/yaml.v2"
)

type proposal struct {
	Title             string   `yaml:"title"`
	Authors           []string `yaml:,flow`
	OwningSIG         string   `yaml:"owning-sig"`
	ParticipatingSIGs []string `yaml:"participating-sigs",flow`
	Reviewers         []string `yaml:,flow`
	Approvers         []string `yaml:,flow`
	Editor            string   `yaml:"editor"`
	CreationDate      string   `yaml:"creation-date"`
	LastUpdated       string   `yaml:"last-updated"`
	Status            string   `yaml:"status"`
	SeeAlso           []string `yaml:"see-also"`
	Replaces          []string `yaml:"replaces"`
	SupersededBy      []string `yaml:"superseded-by"`
}

// YAMLDoc is entirely for testing purposes
func (p *proposal) YAMLDoc() []byte {
	t, err := template.New("yaml").Parse(`title: {{.Title}}
authors:
  {{- range .Authors }}
  - "{{.}}"
  {{- end }}
owning-sig: {{ .OwningSIG }}
{{- if .ParticipatingSIGs }}
participating-sigs:
  {{- range .ParticipatingSIGs }}
  - "{{.}}"
  {{- end }}
{{- end }}
reviewers:
  {{- range .Reviewers }}
  - "{{.}}"
  {{- end }}
approvers:
  {{- range .Approvers }}
  - "{{.}}"
  {{- end }}
{{- if .Editor }}
editor: {{ .Editor }}
{{- end }}
creation-date: {{ .CreationDate }}
last-updated: {{ .LastUpdated }}
status: {{ .Status }}
{{- if .SeeAlso }}
see-also:
  {{- range .SeeAlso }}
  - "{{.}}"
  {{- end }}
{{- end }}
{{- if .Replaces }}
replaces:
  {{- range .Replaces }}
  - "{{.}}"
  {{- end }}
{{- end }}
{{- if .SupersededBy }}
superseded-by:
  {{- range .SupersededBy }}
  - "{{.}}"
  {{- end }}
{{- end }}
`)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, p); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestUnmarshalSuccess(t *testing.T) {
	yamlDoc := &proposal{
		Title:        "test",
		Authors:      []string{"test", "test", "test"},
		Reviewers:    []string{"my reviewer"},
		OwningSIG:    "sig-architecture",
		Status:       "provisional",
		Approvers:    []string{"my approvers"},
		LastUpdated:  "at some point",
		CreationDate: "a while ago",
	}
	p := map[interface{}]interface{}{}

	if err := yaml.Unmarshal(yamlDoc.YAMLDoc(), p); err != nil {
		t.Fatal(err)
	}
	if err := ValidateStructure(p); err != nil {
		t.Fatal(err)
	}
}

func TestValidateStructureSuccess(t *testing.T) {
	testcases := []struct {
		name  string
		input map[interface{}]interface{}
	}{
		{
			name: "Participating sigs is optional",
			input: map[interface{}]interface{}{
				"participating-sigs": "",
			},
		},
		{
			name: "replaces can be empty",
			input: map[interface{}]interface{}{
				"replaces": nil,
			},
		},
		{
			name: "editor can be empty",
			input: map[interface{}]interface{}{
				"editor": nil,
			},
		},
		{
			name: "editor can have a value",
			input: map[interface{}]interface{}{
				"editor": "hello",
			},
		},
		{
			name: "see also can be an empty list",
			input: map[interface{}]interface{}{
				"see-also": []interface{}{},
			},
		},
		{
			name: "superseded-by can be nil",
			input: map[interface{}]interface{}{
				"superseded-by": nil,
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			// add required fields
			tc.input["title"] = "this is a title"
			tc.input["owning-sig"] = "sig-architecture"

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
			name: "Non string key",
			input: map[interface{}]interface{}{
				1: "hello",
			},
		},
		{
			name: "empty title",
			input: map[interface{}]interface{}{
				"title": "",
			},
		},
		{
			name: "non string title",
			input: map[interface{}]interface{}{
				"title": 3,
			},
		},
		{
			name: "title is a list",
			input: map[interface{}]interface{}{
				"title": []interface{}{1, 2, 3},
			},
		},
		{
			name: "authors is an empty list",
			input: map[interface{}]interface{}{
				"authors": []interface{}{},
			},
		},
		{
			name: "authors is just one string",
			input: map[interface{}]interface{}{
				"authors": "me",
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
