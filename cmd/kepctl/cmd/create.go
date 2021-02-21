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

	"k8s.io/enhancements/pkg/proposal"
	"k8s.io/enhancements/pkg/repo"
)

// TODO: Struct literal instead?
var createOpts = proposal.CreateOpts{}

var createCmd = &cobra.Command{
	Use:           "create [KEP]",
	Short:         "Create a new KEP",
	Long:          "Create a new KEP using the current KEP template for the given type",
	Example:       `  kepctl create sig-architecture/000-mykep`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return createOpts.Validate(args)
	},
	RunE: func(*cobra.Command, []string) error {
		return runCreate(&createOpts)
	},
}

func init() {
	// TODO: Should these all be global args?
	createCmd.PersistentFlags().StringVar(
		&createOpts.Title,
		"title",
		"",
		"KEP Title",
	)

	createCmd.PersistentFlags().StringArrayVar(
		&createOpts.Authors,
		"authors",
		[]string{},
		"Authors",
	)

	createCmd.PersistentFlags().StringArrayVar(
		&createOpts.Reviewers,
		"reviewers",
		[]string{},
		"Reviewers",
	)

	createCmd.PersistentFlags().StringVar(
		&createOpts.Type,
		"type",
		"feature",
		"KEP Type",
	)

	createCmd.PersistentFlags().StringVarP(
		&createOpts.State,
		"state",
		"s",
		"provisional",
		"KEP State",
	)

	createCmd.PersistentFlags().StringArrayVar(
		&createOpts.SIGS,
		"sigs",
		[]string{},
		"Participating SIGs",
	)

	createCmd.PersistentFlags().StringArrayVar(
		&createOpts.PRRApprovers,
		"prr-approver",
		[]string{},
		"PRR Approver",
	)

	rootCmd.AddCommand(createCmd)
}

func runCreate(opts *proposal.CreateOpts) error {
	rc, err := repo.New(opts.RepoOpts.RepoPath)
	if err != nil {
		return errors.Wrap(err, "creating repo client")
	}

	return proposal.Create(rc, opts)
}
