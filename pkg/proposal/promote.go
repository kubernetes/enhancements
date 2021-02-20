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
)

type PromoteOpts struct {
	CommonArgs
	Release string
	Stage   string
}

// Validate checks the args provided to the promote command populates the promote opts
func (c *PromoteOpts) Validate(args []string) error {
	return c.validateAndPopulateKEP(args)
}

// Promote changes the stage and target release for a specified KEP based on the
// values specified in PromoteOpts is used to populate the template
func (c *Client) Promote(opts *PromoteOpts) error {
	fmt.Fprintf(c.Out, "Updating KEP %s/%s\n", opts.SIG, opts.Name)

	repoPath, err := c.findEnhancementsRepo(&opts.CommonArgs)
	if err != nil {
		return fmt.Errorf("unable to promote KEP: %s", err)
	}

	p, err := c.readKEP(repoPath, opts.SIG, opts.Name)
	if err != nil {
		return fmt.Errorf("unable to load KEP for promotion: %s", err)
	}

	p.Stage = opts.Stage
	p.LatestMilestone = opts.Release
	p.LastUpdated = opts.Release

	err = c.writeKEP(p, &opts.CommonArgs)
	if err != nil {
		return fmt.Errorf("unable to write updated KEP: %s", err)
	}

	// TODO: Implement ticketing workflow artifact generation
	fmt.Fprintf(c.Out, "KEP %s/%s updated\n", opts.SIG, opts.Name)

	return nil
}
