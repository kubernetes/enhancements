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

package commands

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"k8s.io/enhancements/pkg/proposal"
	"k8s.io/enhancements/pkg/repo"
)

func addPromote(topLevel *cobra.Command) {
	po := proposal.PromoteOpts{}

	cmd := &cobra.Command{
		Use:           "promote [KEP]",
		Short:         "Promote a KEP",
		Long:          "Promote a KEP to a new stage for a target release",
		Example:       `  kepctl promote sig-architecture/000-mykep --stage beta --release v1.20`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return po.Validate(args)
		},
		RunE: func(*cobra.Command, []string) error {
			return runPromote(&po)
		},
	}

	// TODO: Should these all be global args?
	cmd.PersistentFlags().StringVarP(
		&po.Stage,
		"stage",
		"s",
		"",
		"KEP Stage",
	)

	cmd.PersistentFlags().StringVarP(
		&po.Release,
		"release",
		"r",
		"",
		"Target Release",
	)

	topLevel.AddCommand(cmd)
}

func runPromote(opts *proposal.PromoteOpts) error {
	rc, err := repo.New(rootOpts.RepoPath)
	if err != nil {
		return errors.Wrap(err, "creating repo client")
	}

	opts.Repo = rc

	return proposal.Promote(opts)
}
