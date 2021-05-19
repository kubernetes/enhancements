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
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"k8s.io/enhancements/pkg/output"
	"k8s.io/enhancements/pkg/repo"
)

func addQuery(topLevel *cobra.Command) {
	qo := repo.QueryOpts{}

	cmd := &cobra.Command{
		Use:           "query",
		Short:         "Query KEPs",
		Long:          "Query the local filesystem, and optionally GitHub PRs for KEPs",
		Example:       `  kepctl query --sig architecture --status provisional --include-prs`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE: func(*cobra.Command, []string) error {
			return qo.Validate()
		},
		RunE: func(*cobra.Command, []string) error {
			return runQuery(&qo)
		},
	}

	// TODO: Should these all be global args?
	cmd.PersistentFlags().StringSliceVar(
		&qo.Groups,
		"sig",
		nil,
		"SIG. If not specified, KEPs from all SIGs are shown.",
	)

	cmd.PersistentFlags().StringSliceVar(
		&qo.Status,
		"status",
		nil,
		"Status",
	)

	cmd.PersistentFlags().StringSliceVar(
		&qo.Stage,
		"stage",
		nil,
		"Stage",
	)

	cmd.PersistentFlags().StringSliceVar(
		&qo.PRRApprover,
		"prr",
		nil,
		"Prod Readiness Approver",
	)

	cmd.PersistentFlags().StringSliceVar(
		&qo.Approver,
		"approver",
		nil,
		"Approver",
	)

	cmd.PersistentFlags().StringSliceVar(
		&qo.Author,
		"author",
		nil,
		"Author",
	)

	cmd.PersistentFlags().BoolVar(
		&qo.IncludePRs,
		"include-prs",
		false,
		"Include PRs in the results",
	)

	cmd.PersistentFlags().StringVar(
		&qo.Output,
		"output",
		output.DefaultFormat,
		fmt.Sprintf("Output format. Can be %v", output.ValidFormats()),
	)

	topLevel.AddCommand(cmd)
}

func runQuery(opts *repo.QueryOpts) error {
	rc, err := repo.New(rootOpts.RepoPath)
	if err != nil {
		return errors.Wrap(err, "creating repo client")
	}
	rc.TokenPath = rootOpts.TokenPath

	return rc.Query(opts)
}
