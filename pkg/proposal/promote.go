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

	"k8s.io/enhancements/pkg/repo"
)

type PromoteOpts struct {
	Repo *repo.Repo

	// Proposal options
	KEP    string // KEP name sig-xxx/xxx-name
	Name   string
	Number string
	SIG    string

	// Promotion options
	Release string
	Stage   string
}

// Validate checks the args provided to the promote command populates the promote opts
func (o *PromoteOpts) Validate(args []string) error {
	// TODO: Populate logic
	// nolint:gocritic
	return nil // o.Repo.ValidateAndPopulateKEP(args)
}

// Promote changes the stage and target release for a specified KEP based on the
// values specified in PromoteOpts is used to populate the template
func Promote(opts *PromoteOpts) error {
	r := opts.Repo

	fmt.Fprintf(r.Out, "Updating KEP %s/%s\n", opts.SIG, opts.Name)

	p, err := r.ReadKEP(opts.SIG, opts.Name)
	if err != nil {
		return fmt.Errorf("unable to load KEP for promotion: %s", err)
	}

	p.Stage = opts.Stage
	p.LatestMilestone = opts.Release
	p.LastUpdated = opts.Release

	err = r.WriteKEP(p)
	if err != nil {
		return fmt.Errorf("unable to write updated KEP: %s", err)
	}

	// TODO: Implement ticketing workflow artifact generation
	fmt.Fprintf(r.Out, "KEP %s/%s updated\n", opts.SIG, opts.Name)

	return nil
}
