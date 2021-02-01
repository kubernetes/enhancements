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

package api

import (
	"bufio"
	"bytes"
	"io"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type PRRApprovals []*PRRApproval

func (p *PRRApprovals) AddPRRApproval(prrApproval *PRRApproval) {
	*p = append(*p, prrApproval)
}

type PRRApproval struct {
	Number string       `json:"kep-number" yaml:"kep-number"`
	Alpha  PRRMilestone `json:"alpha" yaml:"alpha,omitempty"`
	Beta   PRRMilestone `json:"beta" yaml:"beta,omitempty"`
	Stable PRRMilestone `json:"stable" yaml:"stable,omitempty"`

	// TODO(api): Move to separate struct for handling document parsing
	Error error `json:"-" yaml:"-"`
}

func (prr *PRRApproval) Validate() error {
	v := validator.New()
	if err := v.Struct(prr); err != nil {
		return errors.Wrap(err, "running validation")
	}

	return nil
}

func (prr *PRRApproval) ApproverForStage(stage string) string {
	switch stage {
	case "alpha":
		return prr.Alpha.Approver
	case "beta":
		return prr.Beta.Approver
	case "stable":
		return prr.Stable.Approver
	}

	return ""
}

// TODO(api): Can we refactor the proposal `Milestone` to retrieve this?
type PRRMilestone struct {
	Approver string `json:"approver" yaml:"approver"`
}

type PRRHandler Parser

// TODO(api): Make this a generic parser for all `Document` types
func (p *PRRHandler) Parse(in io.Reader) (*PRRApproval, error) {
	scanner := bufio.NewScanner(in)
	var body bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		body.WriteString(line)
	}

	approval := &PRRApproval{}
	if err := scanner.Err(); err != nil {
		return approval, errors.Wrap(err, "reading file")
	}

	if err := yaml.Unmarshal(body.Bytes(), &approval); err != nil {
		p.Errors = append(p.Errors, errors.Wrap(err, "error unmarshalling YAML"))
		return approval, errors.Wrap(err, "unmarshalling YAML")
	}

	if valErr := approval.Validate(); valErr != nil {
		p.Errors = append(p.Errors, errors.Wrap(valErr, "validating PRR"))
		return approval, errors.Wrap(valErr, "validating PRR")
	}

	return approval, nil
}
