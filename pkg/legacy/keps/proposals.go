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
	"gopkg.in/yaml.v3"

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/legacy/keps/validations"
)

type Parser struct {
	Groups       []string
	PRRApprovers []string
}

func (p *Parser) Parse(in io.Reader) *api.Proposal {
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
	proposal := &api.Proposal{
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

	if err := validations.ValidateStructure(p.Groups, p.PRRApprovers, test); err != nil {
		proposal.Error = errors.Wrap(err, "error validating KEP metadata")
		return proposal
	}

	proposal.Error = yaml.Unmarshal(metadata, proposal)
	proposal.ID = hash(proposal.OwningSIG + ":" + proposal.Title)
	return proposal
}

func hash(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
