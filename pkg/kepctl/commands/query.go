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

	"k8s.io/enhancements/pkg/repo"
)

// TODO: Struct literal instead?
var queryOpts = repo.QueryOpts{}

var queryCmd = &cobra.Command{
	Use:           "query",
	Short:         "Query KEPs",
	Long:          "Query the local filesystem, and optionally GitHub PRs for KEPs",
	Example:       `  kepctl query --sig architecture --status provisional --include-prs`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PreRunE: func(*cobra.Command, []string) error {
		return queryOpts.Validate()
	},
	RunE: func(*cobra.Command, []string) error {
		return runQuery(&queryOpts)
	},
}

func init() {
	// TODO: Should these all be global args?
	queryCmd.PersistentFlags().StringSliceVar(
		&queryOpts.SIG,
		"sig",
		nil,
		"SIG. If not specified, KEPs from all SIGs are shown.",
	)

	queryCmd.PersistentFlags().StringSliceVar(
		&queryOpts.Status,
		"status",
		nil,
		"Status",
	)

	queryCmd.PersistentFlags().StringSliceVar(
		&queryOpts.Stage,
		"stage",
		nil,
		"Stage",
	)

	queryCmd.PersistentFlags().StringSliceVar(
		&queryOpts.PRRApprover,
		"prr",
		nil,
		"Prod Readiness Approver",
	)

	queryCmd.PersistentFlags().StringSliceVar(
		&queryOpts.Approver,
		"approver",
		nil,
		"Approver",
	)

	queryCmd.PersistentFlags().StringSliceVar(
		&queryOpts.Author,
		"author",
		nil,
		"Author",
	)

	queryCmd.PersistentFlags().BoolVar(
		&queryOpts.IncludePRs,
		"include-prs",
		false,
		"Include PRs in the results",
	)

	queryCmd.PersistentFlags().StringVar(
		&queryOpts.Output,
		"output",
		repo.DefaultOutputOpt,
		fmt.Sprintf(
			"Output format. Can be %v", repo.SupportedOutputOpts,
		),
	)

	rootCmd.AddCommand(queryCmd)
}

func runQuery(opts *repo.QueryOpts) error {
	rc, err := repo.New(opts.RepoOpts.RepoPath)
	if err != nil {
		return errors.Wrap(err, "creating repo client")
	}

	return rc.Query(opts)
}
