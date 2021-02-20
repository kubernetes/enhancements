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

package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"k8s.io/enhancements/pkg/kepctl"
)

// TODO: Struct literal instead?
var promoteOpts = kepctl.PromoteOpts{}

var promoteCmd = &cobra.Command{
	Use:           "promote [KEP]",
	Short:         "Promote a KEP",
	Long:          "Promote a KEP to a new stage for a target release",
	Example:       `  kepctl promote sig-architecture/000-mykep --stage beta --release v1.20`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return promoteOpts.Validate(args)
	},
	RunE: func(*cobra.Command, []string) error {
		return runPromote(&promoteOpts)
	},
}

func init() {
	// TODO: Should these all be global args?
	promoteCmd.PersistentFlags().StringVarP(
		&promoteOpts.Stage,
		"stage",
		"s",
		"",
		"KEP Stage",
	)

	promoteCmd.PersistentFlags().StringVarP(
		&promoteOpts.Release,
		"release",
		"r",
		"",
		"Target Release",
	)

	rootCmd.AddCommand(promoteCmd)
}

func runPromote(opts *kepctl.PromoteOpts) error {
	k, err := kepctl.New(opts.RepoPath)
	if err != nil {
		return errors.Wrap(err, "creating kepctl client")
	}

	return k.Promote(opts)
}
