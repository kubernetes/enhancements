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

type Proposal struct {
	Title             string   `yaml:"title"`
	Authors           []string `yaml:,flow`
	OwningSIG         string   `yaml:"owning-sig"`
	ParticipatingSIGs []string `yaml:"participating-sigs",flow,omitempty`
	Reviewers         []string `yaml:,flow`
	Approvers         []string `yaml:,flow`
	Editor            string   `yaml:"editor,omitempty"`
	CreationDate      string   `yaml:"creation-date"`
	LastUpdated       string   `yaml:"last-updated"`
	Status            string   `yaml:"status"`
	SeeAlso           []string `yaml:"see-also,omitempty"`
	Replaces          []string `yaml:"replaces,omitempty"`
	SupersededBy      []string `yaml:"superseded-by,omitempty"`

	Filename string `yaml:"-"`
	Error    error  `yaml:"-"`
	Contents string `yaml:"-"`
}

type Parser struct{}

func (p *Parser) Parse(in io.Reader) *Proposal {
	scanner := bufio.NewScanner(in)
	count := 0
	metadata := []byte{}
	var body bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		body.WriteString(line)
		if count == 2 {
			continue
		}
		if strings.Contains(line, "---") {
			count++
			continue
		}
		if count == 1 {
			metadata = append(metadata, []byte(line)...)
		}
	}
	proposal := &Proposal{
		Contents: body.String(),
	}
	if err := scanner.Err(); err != nil {
		proposal.Error = errors.Wrap(err, "error reading file")
		return proposal
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
	return proposal
}
