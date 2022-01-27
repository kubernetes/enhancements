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

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/proposal"
	"k8s.io/enhancements/pkg/repo"
)

func addCreate(topLevel *cobra.Command) {
	co := proposal.CreateOpts{}

	cmd := &cobra.Command{
		Use:           "create",
		Short:         "Create a new KEP",
		Long:          "Create a new KEP using the current KEP template for the given type",
		Example:       `  kepctl create --name a-path --title "My KEP" --number 123 --owning-sig sig-foo`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return co.Validate(args)
		},
		RunE: func(*cobra.Command, []string) error {
			return runCreate(&co)
		},
	}

	// TODO: Should these all be global args?
	cmd.PersistentFlags().StringVar(
		&co.Title,
		"title",
		"",
		"KEP Title",
	)

	cmd.PersistentFlags().StringVar(
		&co.Number,
		"number",
		"",
		"Number",
	)

	cmd.PersistentFlags().StringVar(
		&co.Name,
		"name",
		"",
		"Name",
	)

	cmd.PersistentFlags().StringArrayVar(
		&co.Authors,
		"authors",
		[]string{},
		"Authors",
	)

	cmd.PersistentFlags().StringArrayVar(
		&co.Reviewers,
		"reviewers",
		[]string{},
		"Reviewers",
	)

	cmd.PersistentFlags().StringVar(
		&co.Type,
		"type",
		"feature",
		"KEP Type",
	)

	cmd.PersistentFlags().StringVarP(
		&co.State,
		"state",
		"s",
		string(api.ProvisionalStatus),
		"KEP State",
	)

	cmd.PersistentFlags().StringVar(
		&co.SIG,
		"owning-sig",
		"",
		"Owning SIG",
	)

	cmd.PersistentFlags().StringArrayVar(
		&co.ParticipatingSIGs,
		"participating-sigs",
		[]string{},
		"Participating SIGs",
	)

	cmd.PersistentFlags().StringArrayVar(
		&co.PRRApprovers,
		"prr-approver",
		[]string{},
		"PRR Approver",
	)

	topLevel.AddCommand(cmd)
}

func runCreate(opts *proposal.CreateOpts) error {
	rc, err := repo.New(rootOpts.RepoPath)
	if err != nil {
		return errors.Wrap(err, "creating repo client")
	}

	opts.Repo = rc

	return proposal.Create(opts)
}
