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
	"crypto/md5"
	"fmt"
	"io"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"

	"k8s.io/enhancements/pkg/yaml"
)

type Stage string

const (
	AlphaStage  Stage = "alpha"
	BetaStage   Stage = "beta"
	StableStage Stage = "stable"
)

var ValidStages = []Stage{
	AlphaStage,
	BetaStage,
	StableStage,
}

func (s Stage) IsValid() error {
	for _, s2 := range ValidStages {
		if s == s2 {
			return nil
		}
	}
	return fmt.Errorf("invalid stage: %v, should be one of %v", s, ValidStages)
}

type Status string

const (
	ProvisionalStatus   Status = "provisional"
	ImplementableStatus Status = "implementable"
	ImplementedStatus   Status = "implemented"
	DeferredStatus      Status = "deferred"
	RejectedStatus      Status = "rejected"
	WithdrawnStatus     Status = "withdrawn"
	ReplacedStatus      Status = "replaced"
)

var ValidStatuses = []Status{
	ProvisionalStatus,
	ImplementableStatus,
	ImplementedStatus,
	DeferredStatus,
	RejectedStatus,
	WithdrawnStatus,
	ReplacedStatus,
}

func (s Status) IsValid() error {
	for _, s2 := range ValidStatuses {
		if s == s2 {
			return nil
		}
	}
	return fmt.Errorf("invalid status: %v, should be one of %v", s, ValidStatuses)
}

type Proposals []*Proposal

func (p *Proposals) AddProposal(proposal *Proposal) {
	*p = append(*p, proposal)
}

// TODO(api): json fields are not using consistent casing
type Proposal struct {
	ID       string `json:"id"`
	PRNumber string `json:"prNumber,omitempty"`
	Name     string `json:"name,omitempty"`

	Title             string   `json:"title" yaml:"title" validate:"required"`
	Number            string   `json:"kepNumber" yaml:"kep-number" validate:"required"`
	Authors           []string `json:"authors" yaml:",flow" validate:"required"`
	OwningSIG         string   `json:"owningSig" yaml:"owning-sig" validate:"required"`
	ParticipatingSIGs []string `json:"participatingSigs" yaml:"participating-sigs,flow,omitempty"`
	Reviewers         []string `json:"reviewers" yaml:",flow"`
	Approvers         []string `json:"approvers" yaml:",flow" validate:"required"`
	PRRApprovers      []string `json:"prrApprovers" yaml:"prr-approvers,flow"`
	Editor            string   `json:"editor" yaml:"editor,omitempty"`
	CreationDate      string   `json:"creationDate" yaml:"creation-date"`
	LastUpdated       string   `json:"lastUpdated" yaml:"last-updated"`
	Status            Status   `json:"status" yaml:"status" validate:"required"`
	SeeAlso           []string `json:"seeAlso" yaml:"see-also,omitempty"`
	Replaces          []string `json:"replaces" yaml:"replaces,omitempty"`
	SupersededBy      []string `json:"supersededBy" yaml:"superseded-by,omitempty"`

	Stage           Stage     `json:"stage" yaml:"stage"`
	LatestMilestone string    `json:"latestMilestone" yaml:"latest-milestone"`
	Milestone       Milestone `json:"milestone" yaml:"milestone"`

	FeatureGates     []FeatureGate `json:"featureGates" yaml:"feature-gates"`
	DisableSupported bool          `json:"disableSupported" yaml:"disable-supported"`
	Metrics          []string      `json:"metrics" yaml:"metrics"`

	Filename string `json:"-" yaml:"-"`
	Error    error  `json:"-" yaml:"-"`
	Contents string `json:"markdown" yaml:"-"`
}

func (p *Proposal) IsMissingMilestone() bool {
	return p.LatestMilestone == ""
}

func (p *Proposal) IsMissingStage() bool {
	return p.Stage == ""
}

type Milestone struct {
	Alpha      string `json:"alpha" yaml:"alpha"`
	Beta       string `json:"beta" yaml:"beta"`
	Stable     string `json:"stable" yaml:"stable"`
	Deprecated string `json:"deprecated" yaml:"deprecated,omitempty"`
	Removed    string `json:"removed" yaml:"removed,omitempty"`
}

type FeatureGate struct {
	Name       string   `json:"name" yaml:"name"`
	Components []string `json:"components" yaml:"components"`
}
type KEPHandler Parser

// TODO(api): Make this a generic parser for all `Document` types
func (k *KEPHandler) Parse(in io.Reader) (*Proposal, error) {
	scanner := bufio.NewScanner(in)
	count := 0
	metadata := []byte{}
	var body bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		if strings.Contains(line, "---") {
			count++
			continue
		}
		if count == 1 {
			metadata = append(metadata, []byte(line)...)
		} else {
			body.WriteString(line)
		}
	}

	kep := &Proposal{
		Contents: body.String(),
	}

	if err := scanner.Err(); err != nil {
		return kep, errors.Wrap(err, "reading file")
	}

	// this file is just the KEP metadata
	if count == 0 {
		metadata = body.Bytes()
		kep.Contents = ""
	}

	if err := yaml.UnmarshalStrict(metadata, &kep); err != nil {
		k.Errors = append(k.Errors, errors.Wrap(err, "error unmarshalling YAML"))
		return kep, errors.Wrap(err, "unmarshalling YAML")
	}

	if err := k.validateStruct(kep); err != nil {
		k.Errors = append(k.Errors, err)
		return kep, fmt.Errorf("validating KEP: %w", err)
	}

	kep.ID = hash(kep.OwningSIG + ":" + kep.Title)

	return kep, nil
}

// validateStruct returns an error if the given Proposal has invalid fields
// as defined by struct tags, or nil if there are no invalid fields
func (k *KEPHandler) validateStruct(p *Proposal) error {
	v := validator.New()
	return v.Struct(p)
}

// validateGroups returns errors for each invalid group (e.g. SIG) in the given
// Proposal, or nil if there are no invalid groups
func (k *KEPHandler) validateGroups(p *Proposal) []error {
	var errs []error
	validGroups := make(map[string]bool)
	for _, g := range k.Groups {
		validGroups[g] = true
	}
	for _, g := range p.ParticipatingSIGs {
		if _, ok := validGroups[g]; !ok {
			errs = append(errs, fmt.Errorf("invalid participating-sig: %s", g))
		}
	}
	if _, ok := validGroups[p.OwningSIG]; !ok {
		errs = append(errs, fmt.Errorf("invalid owning-sig: %s", p.OwningSIG))
	}
	return errs
}

// validatePRRApprovers returns errors for each invalid PRR Approver in the
// given Proposal, or nil if there are no invalid PRR Approvers
func (k *KEPHandler) validatePRRApprovers(p *Proposal) []error {
	var errs []error
	validPRRApprovers := make(map[string]bool)
	for _, a := range k.PRRApprovers {
		validPRRApprovers[a] = true
	}
	for _, a := range p.PRRApprovers {
		if _, ok := validPRRApprovers[a]; !ok {
			errs = append(errs, fmt.Errorf("invalid prr-approver: %s", a))
		}
	}
	return errs
}

// Validate returns errors for each reason the given proposal is invalid,
// or nil if it is valid
func (k *KEPHandler) Validate(p *Proposal) []error {
	var allErrs []error

	if err := k.validateStruct(p); err != nil {
		allErrs = append(allErrs, fmt.Errorf("struct-based validation: %w", err))
	}
	if errs := k.validateGroups(p); errs != nil {
		allErrs = append(allErrs, errs...)
	}
	if errs := k.validatePRRApprovers(p); errs != nil {
		allErrs = append(allErrs, errs...)
	}
	if err := p.Status.IsValid(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := p.Stage.IsValid(); err != nil {
		allErrs = append(allErrs, err)
	}
	if p.Status == ImplementedStatus && p.Stage != StableStage {
		allErrs = append(allErrs, fmt.Errorf("status:implemented implies stage:stable but found: %v", p.Stage))
	}
	return allErrs
}

func hash(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
