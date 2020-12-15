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

package keps

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"k8s.io/enhancements/pkg/kepval/keps/validations"
)

type Proposals []*Proposal

func (p *Proposals) AddProposal(proposal *Proposal) {
	*p = append(*p, proposal)
}

type Milestone struct {
	Alpha  string `json:"alpha" yaml:"alpha"`
	Beta   string `json:"beta" yaml:"beta"`
	Stable string `json:"stable" yaml:"stable"`
}

type FeatureGate struct {
	Name       string   `json:"name" yaml:"name"`
	Components []string `json:"components" yaml:"components"`
}

type Proposal struct {
	ID       string `json:"id"`
	PRNumber string `json:"prNumber,omitempty"`
	Name     string `json:"name,omitempty"`

	Title             string   `json:"title" yaml:"title"`
	Number            string   `json:"kep-number" yaml:"kep-number"`
	Authors           []string `json:"authors" yaml:",flow"`
	OwningSIG         string   `json:"owningSig" yaml:"owning-sig"`
	ParticipatingSIGs []string `json:"participatingSigs" yaml:"participating-sigs,flow,omitempty"`
	Reviewers         []string `json:"reviewers" yaml:",flow"`
	Approvers         []string `json:"approvers" yaml:",flow"`
	PRRApprovers      []string `json:"prrApprovers" yaml:"prr-approvers,flow"`
	Editor            string   `json:"editor" yaml:"editor,omitempty"`
	CreationDate      string   `json:"creationDate" yaml:"creation-date"`
	LastUpdated       string   `json:"lastUpdated" yaml:"last-updated"`
	Status            string   `json:"status" yaml:"status"`
	SeeAlso           []string `json:"seeAlso" yaml:"see-also,omitempty"`
	Replaces          []string `json:"replaces" yaml:"replaces,omitempty"`
	SupersededBy      []string `json:"supersededBy" yaml:"superseded-by,omitempty"`

	Stage           string    `json:"stage" yaml:"stage"`
	LatestMilestone string    `json:"latestMilestone" yaml:"latest-milestone"`
	Milestone       Milestone `json:"milestone" yaml:"milestone"`

	FeatureGates     []FeatureGate `json:"featureGates" yaml:"feature-gates"`
	DisableSupported bool          `json:"disableSupported" yaml:"disable-supported"`
	Metrics          []string      `json:"metrics" yaml:"metrics"`

	Error    error  `json:"-" yaml:"-"`
	Contents string `json:"markdown" yaml:"-"`
}

type Parser struct{}

func (p *Parser) Parse(in io.Reader) *Proposal {
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
	proposal := &Proposal{
		Contents: body.String(),
	}
	if err := scanner.Err(); err != nil {
		proposal.Error = errors.Wrap(err, "error reading file")
		return proposal
	}

	// this file is just the KEP metadata
	if count == 0 {
		metadata = body.Bytes()
		proposal.Contents = ""
	}

	// First do structural checks
	test := map[interface{}]interface{}{}
	if err := yaml.Unmarshal(metadata, test); err != nil {
		proposal.Error = errors.Wrap(err, "error unmarshaling YAML")
		return proposal
	}
	if err := validations.ValidateStructure(test); err != nil {
		proposal.Error = errors.Wrap(err, "error validating KEP metadata")
		return proposal
	}

	proposal.Error = yaml.UnmarshalStrict(metadata, proposal)
	proposal.ID = hash(proposal.OwningSIG + ":" + proposal.Title)
	return proposal
}

func hash(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
