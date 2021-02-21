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

package proposal

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/repo"
)

type CreateOpts struct {
	RepoOpts     *repo.Options
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
	repoOpts := c.RepoOpts
	err := repoOpts.ValidateAndPopulateKEP(args)
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
func Create(rc *repo.Client, opts *CreateOpts) error {
	repoOpts := opts.RepoOpts

	fmt.Fprintf(rc.Out, "Creating KEP %s %s %s\n", repoOpts.SIG, repoOpts.Number, repoOpts.Name)

	repoPath, err := rc.FindEnhancementsRepo(repoOpts)
	fmt.Fprintf(rc.Out, "Looking for enhancements repo in %s\n", repoPath)
	if err != nil {
		return fmt.Errorf("unable to create KEP: %s", err)
	}

	t, err := rc.GetKepTemplate(repoPath)
	if err != nil {
		return err
	}

	updateTemplate(t, opts)

	err = validateKEP(t)
	if err != nil {
		return err
	}

	err = createKEP(rc, t, opts)
	if err != nil {
		return err
	}

	return nil
}

func createKEP(rc *repo.Client, kep *api.Proposal, opts *CreateOpts) error {
	args := opts.RepoOpts

	fmt.Fprintf(rc.Out, "Generating new KEP %s in %s ===>\n", args.Name, args.SIG)

	err := rc.WriteKEP(kep, args)
	if err != nil {
		return fmt.Errorf("unable to create KEP: %s", err)
	}

	path, err := rc.FindEnhancementsRepo(args)
	if err != nil {
		return err
	}

	b, err := rc.GetReadmeTemplate(path)
	if err != nil {
		return fmt.Errorf("couldn't find README template: %s", err)
	}

	newPath := filepath.Join(path, "keps", args.SIG, args.Name, "README.md")
	if writeErr := ioutil.WriteFile(newPath, b, os.ModePerm); writeErr != nil {
		return errors.Wrapf(writeErr, "writing KEP data to file")
	}

	return nil
}

func updateTemplate(t *api.Proposal, opts *CreateOpts) {
	repoOpts := opts.RepoOpts

	if opts.State != "" {
		t.Status = opts.State
	}

	now := time.Now()
	layout := "2006-01-02"

	t.CreationDate = now.Format(layout)
	t.Number = repoOpts.Number
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

	t.OwningSIG = repoOpts.SIG

	// TODO(lint): appendAssign: append result not assigned to the same slice (gocritic)
	//nolint:gocritic
	t.ParticipatingSIGs = append(opts.SIGS, repoOpts.SIG)
	t.Filename = repoOpts.Name
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

func validateKEP(p *api.Proposal) error {
	b, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	r := bytes.NewReader(b)
	parser, err := api.NewKEPHandler()
	if err != nil {
		return errors.Wrap(err, "creating KEP handler")
	}

	kep, err := parser.Parse(r)
	if err != nil {
		return fmt.Errorf("kep is invalid: %s", kep.Error)
	}

	return nil
}
