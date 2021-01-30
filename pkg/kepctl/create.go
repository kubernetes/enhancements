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

package kepctl

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/enhancements/api"
)

type CreateOpts struct {
	CommonArgs
	Title        string
	Approvers    []string
	Authors      []string
	Reviewers    []string
	Type         string
	State        string
	SIGS         []string
	PRRApprovers []string
}

// Validate checks the args provided to the create command and parses the sig,
// kep # and name to populate the create opts
func (c *CreateOpts) Validate(args []string) error {
	err := c.validateAndPopulateKEP(args)
	if err != nil {
		return err
	}
	if len(c.PRRApprovers) == 0 {
		return errors.New("must provide at least one PRR Approver")
	}
	return nil
}

// Create builds a new KEP based on the README.md and kep.yaml templates in the
// path specified by the command args. CreateOpts is used to populate the template
func (c *Client) Create(opts *CreateOpts) error {
	fmt.Fprintf(c.Out, "Creating KEP %s %s %s\n", opts.SIG, opts.Number, opts.Name)

	repoPath, err := c.findEnhancementsRepo(&opts.CommonArgs)
	fmt.Fprintf(c.Out, "Looking for enhancements repo in %s\n", repoPath)
	if err != nil {
		return fmt.Errorf("unable to create KEP: %s", err)
	}

	t, err := c.getKepTemplate(repoPath)
	if err != nil {
		return err
	}

	updateTemplate(t, opts)

	err = validateKEP(t)
	if err != nil {
		return err
	}

	err = c.createKEP(t, opts)
	if err != nil {
		return err
	}

	return nil
}

func updateTemplate(t *api.Proposal, opts *CreateOpts) {
	if opts.State != "" {
		t.Status = opts.State
	}

	now := time.Now()
	layout := "2006-01-02"

	t.CreationDate = now.Format(layout)
	t.Number = opts.Number
	t.Title = opts.Title

	if len(opts.Authors) > 0 {
		authors := []string{}
		for _, author := range opts.Authors {
			if !strings.HasPrefix(author, "@") {
				author = fmt.Sprintf("@%s", author)
			}

			authors = append(authors, author)
		}

		t.Authors = authors
	}

	if len(opts.Approvers) > 0 {
		t.Approvers = updatePersonReference(opts.Approvers)
	}

	if len(opts.Reviewers) > 0 {
		t.Reviewers = updatePersonReference(opts.Reviewers)
	}

	t.OwningSIG = opts.SIG
	t.ParticipatingSIGs = append(opts.SIGS, opts.SIG)
	t.Filename = opts.Name
	t.LastUpdated = "v1.19"
	if len(opts.PRRApprovers) > 0 {
		t.PRRApprovers = updatePersonReference(opts.PRRApprovers)
	}
}

func updatePersonReference(names []string) []string {
	persons := []string{}
	for _, name := range names {
		if !strings.HasPrefix(name, "@") {
			name = fmt.Sprintf("@%s", name)
		}
		persons = append(persons, name)
	}
	return persons
}

func (c *Client) createKEP(kep *api.Proposal, opts *CreateOpts) error {
	fmt.Fprintf(c.Out, "Generating new KEP %s in %s ===>\n", opts.Name, opts.SIG)

	args := &opts.CommonArgs
	err := c.writeKEP(kep, args)
	if err != nil {
		return fmt.Errorf("unable to create KEP: %s", err)
	}

	path, err := c.findEnhancementsRepo(args)
	if err != nil {
		return err
	}

	b, err := c.getReadmeTemplate(path)
	if err != nil {
		return fmt.Errorf("couldn't find README template: %s", err)
	}

	newPath := filepath.Join(path, "keps", opts.SIG, opts.Name, "README.md")
	ioutil.WriteFile(newPath, b, os.ModePerm)

	return nil
}
