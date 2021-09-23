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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/repo"
)

type CreateOpts struct {
	Repo *repo.Repo

	// Proposal options
	KEP    string // KEP name sig-xxx/xxx-name
	Name   string
	Number string
	SIG    string

	// Create options
	Title             string
	Approvers         []string
	Authors           []string
	Reviewers         []string
	Type              string
	State             string
	ParticipatingSIGs []string
	PRRApprovers      []string
}

// Validate checks the args provided to the create command and parses the sig,
// kep # and name to populate the create opts
func (c *CreateOpts) Validate(args []string) error {
	// TODO: Populate logic
	// nolint:gocritic
	/*
		err := repoOpts.ValidateAndPopulateKEP(args)
		if err != nil {
			return err
		}

		if len(c.PRRApprovers) == 0 {
			return errors.New("must provide at least one PRR Approver")
		}
	*/

	return nil
}

// Create builds a new KEP based on the README.md and kep.yaml templates in the
// path specified by the command args. CreateOpts is used to populate the template
func Create(opts *CreateOpts) error {
	r := opts.Repo

	logrus.Infof("Creating KEP %s %s %s", opts.SIG, opts.Number, opts.Name)

	kep := &api.Proposal{}

	populateProposal(kep, opts)

	errs := r.KEPHandler.Validate(kep)
	if errs != nil {
		return fmt.Errorf("invalid kep: %v", errs)
	}

	err := createKEP(kep, opts)
	if err != nil {
		return err
	}

	return nil
}

func createKEP(kep *api.Proposal, opts *CreateOpts) error {
	r := opts.Repo

	logrus.Infof("Generating new KEP %s in %s ===>", opts.Name, opts.SIG)

	err := r.WriteKEP(kep)
	if err != nil {
		return fmt.Errorf("unable to create KEP: %s", err)
	}

	template := r.ProposalTemplate

	newPath := filepath.Join(
		r.ProposalPath,
		opts.SIG,
		opts.Name,
		repo.ProposalFilename,
	)

	if writeErr := ioutil.WriteFile(newPath, template, os.ModePerm); writeErr != nil {
		return errors.Wrapf(writeErr, "writing KEP data to file")
	}

	return nil
}

func populateProposal(p *api.Proposal, opts *CreateOpts) {
	p.Name = opts.Name

	if opts.State != "" {
		p.Status = api.Status(opts.State)
	}

	now := time.Now()
	layout := "2006-01-02"

	p.CreationDate = now.Format(layout)
	p.Number = opts.Number
	p.Title = opts.Title

	if len(opts.Authors) > 0 {
		authors := []string{}
		for _, author := range opts.Authors {
			if !strings.HasPrefix(author, "@") {
				author = fmt.Sprintf("@%s", author)
			}

			authors = append(authors, author)
		}

		p.Authors = authors
	}

	if len(opts.Approvers) > 0 {
		p.Approvers = updatePersonReference(opts.Approvers)
	}

	if len(opts.Reviewers) > 0 {
		p.Reviewers = updatePersonReference(opts.Reviewers)
	}

	p.OwningSIG = opts.SIG

	// TODO(lint): appendAssign: append result not assigned to the same slice (gocritic)
	//nolint:gocritic
	p.ParticipatingSIGs = append(opts.ParticipatingSIGs, opts.SIG)
	p.Filename = opts.Name
	p.LastUpdated = "v1.19"

	if len(opts.PRRApprovers) > 0 {
		p.PRRApprovers = updatePersonReference(opts.PRRApprovers)
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
