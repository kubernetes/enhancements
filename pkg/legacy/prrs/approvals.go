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
	"gopkg.in/yaml.v3"

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/legacy/prrs/validations"
)

type Parser struct {
	PRRApprovers []string
}

func (p *Parser) Parse(in io.Reader) *api.PRRApproval {
	scanner := bufio.NewScanner(in)
	var body bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text() + "\n"
		body.WriteString(line)
	}

	approval := &api.PRRApproval{}
	if err := scanner.Err(); err != nil {
		approval.Error = errors.Wrap(err, "error reading file")
		return approval
	}

	// First do structural checks
	test := map[interface{}]interface{}{}
	if err := yaml.Unmarshal(body.Bytes(), test); err != nil {
		approval.Error = errors.Wrap(err, "error unmarshalling YAML")
		return approval
	}
	if err := validations.ValidateStructure(p.PRRApprovers, test); err != nil {
		approval.Error = errors.Wrap(err, "error validating PRR approval metadata")
		return approval
	}

	approval.Error = yaml.Unmarshal(body.Bytes(), approval)

	return approval
}
