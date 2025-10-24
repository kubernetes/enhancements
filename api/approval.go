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
	"fmt"
	"io"

	"github.com/go-playground/validator/v10"

	"k8s.io/enhancements/pkg/yaml"
)

type PRRApprovals []*PRRApproval

func (p *PRRApprovals) AddPRRApproval(prrApproval *PRRApproval) {
	*p = append(*p, prrApproval)
}

type PRRApproval struct {
	Number string `json:"kepNumber" yaml:"kep-number" validate:"required"`

	// TODO: Need to validate these milestone pointers are not nil
	Alpha      *PRRMilestone `json:"alpha" yaml:"alpha,omitempty"`
	Beta       *PRRMilestone `json:"beta" yaml:"beta,omitempty"`
	Stable     *PRRMilestone `json:"stable" yaml:"stable,omitempty"`
	Deprecated *PRRMilestone `json:"deprecated" yaml:"deprecated,omitempty"`
	Removed    *PRRMilestone `json:"removed" yaml:"removed,omitempty"`
	Disabled   *PRRMilestone `json:"disabled" yaml:"disabled,omitempty"`

	// TODO(api): Move to separate struct for handling document parsing
	Error error `json:"-" yaml:"-"`
}

func (prr *PRRApproval) Validate() error {
	v := validator.New()
	if err := v.Struct(prr); err != nil {
		return fmt.Errorf("running validation: %w", err)
	}

	return nil
}

func (prr *PRRApproval) ApproverForStage(stage Stage) (string, error) {
	if err := stage.IsValid(); err != nil {
		return "", err
	}

	// KEPs are usually of 2 types:
	// 1. Features that go through Alpha->Beta->Stable
	// 2. Removals that go through Deprecated->Disabled->Removed
	if prr.Alpha == nil && prr.Beta == nil && prr.Stable == nil && prr.Deprecated == nil && prr.Disabled == nil && prr.Removed == nil {
		return "", ErrPRRMilestonesAllEmpty
	}

	switch stage {
	case AlphaStage:
		if prr.Alpha == nil {
			return "", ErrPRRMilestoneIsNil
		}
		return prr.Alpha.Approver, nil

	case BetaStage:
		if prr.Beta == nil {
			return "", ErrPRRMilestoneIsNil
		}
		return prr.Beta.Approver, nil

	case StableStage:
		if prr.Stable == nil {
			return "", ErrPRRMilestoneIsNil
		}
		return prr.Stable.Approver, nil

	case Deprecated:
		if prr.Deprecated == nil {
			return "", ErrPRRMilestoneIsNil
		}
		return prr.Deprecated.Approver, nil

	case Disabled:
		if prr.Disabled == nil {
			return "", ErrPRRMilestoneIsNil
		}
		return prr.Disabled.Approver, nil

	case Removed:
		if prr.Removed == nil {
			return "", ErrPRRMilestoneIsNil
		}
		return prr.Removed.Approver, nil
	}

	return "", ErrPRRApproverUnknown
}

// TODO(api): Can we refactor the proposal `Milestone` to retrieve this?
type PRRMilestone struct {
	Approver string `json:"approver" yaml:"approver" validate:"required"`
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
		return approval, fmt.Errorf("reading file: %w", err)
	}

	if err := yaml.UnmarshalStrict(body.Bytes(), &approval); err != nil {
		p.Errors = append(p.Errors, fmt.Errorf("error unmarshalling YAML: %w", err))
		return approval, fmt.Errorf("unmarshalling YAML: %w", err)
	}

	if valErr := approval.Validate(); valErr != nil {
		p.Errors = append(p.Errors, fmt.Errorf("validating PRR: %w", valErr))
		return approval, fmt.Errorf("validating PRR: %w", valErr)
	}

	return approval, nil
}
