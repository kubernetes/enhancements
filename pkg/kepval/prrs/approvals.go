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
	"bufio"
	"bytes"
	"io"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"k8s.io/enhancements/pkg/kepval/prrs/validations"
)

type Approvals []*Approval

func (a *Approvals) AddApproval(approval *Approval) {
	*a = append(*a, approval)
}

type Milestone struct {
	Approver string `json:"approver" yaml:"approver"`
}

type Approval struct {
	Number string    `json:"kep-number" yaml:"kep-number"`
	Alpha  Milestone `json:"alpha" yaml:"alpha"`
	Beta   Milestone `json:"beta" yaml:"beta"`
	Stable Milestone `json:"stable" yaml:"stable"`

	Error    error  `json:"-" yaml:"-"`
}

type Parser struct{}

func (p *Parser) Parse(in io.Reader) *Approval {
	scanner := bufio.NewScanner(in)
	var body bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		body.WriteString(line)
	}

	approval := &Approval{}
	if err := scanner.Err(); err != nil {
		approval.Error = errors.Wrap(err, "error reading file")
		return approval
	}

	// First do structural checks
	test := map[interface{}]interface{}{}
	if err := yaml.Unmarshal(body.Bytes(), test); err != nil {
		approval.Error = errors.Wrap(err, "error unmarshaling YAML")
		return approval
	}
	if err := validations.ValidateStructure(test); err != nil {
		approval.Error = errors.Wrap(err, "error validating PRR approval metadata")
		return approval
	}

	approval.Error = yaml.UnmarshalStrict(body.Bytes(), approval)
	return approval
}
