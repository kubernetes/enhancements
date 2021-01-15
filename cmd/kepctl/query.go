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

package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"k8s.io/enhancements/pkg/kepctl"
)

func buildQueryCommand(k *kepctl.Client) *cobra.Command {
	opts := kepctl.QueryOpts{}
	cmd := &cobra.Command{
		Use:     "query",
		Short:   "Query KEPs",
		Long:    "Query the local filesystem, and optionally GitHub PRs for KEPs",
		Example: `  kepctl query --sig architecture --status provisional --include-prs`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.Validate(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return k.Query(opts)
		},
	}

	f := cmd.Flags()
	f.StringSliceVar(&opts.SIG, "sig", nil, "SIG")
	f.StringSliceVar(&opts.Status, "status", nil, "Status")
	f.StringSliceVar(&opts.Stage, "stage", nil, "Stage")
	f.StringSliceVar(&opts.PRRApprover, "prr", nil, "Prod Readiness Approver")
	f.StringSliceVar(&opts.Approver, "approver", nil, "Approver")
	f.StringSliceVar(&opts.Author, "author", nil, "Author")
	f.BoolVar(&opts.IncludePRs, "include-prs", false, "Include PRs in the results")
	f.StringVar(&opts.Output, "output", kepctl.DefaultOutputOpt, fmt.Sprintf("Output format. Can be %v", kepctl.SupportedOutputOpts))

	addRepoPathFlag(f, &opts.CommonArgs)

	return cmd
}
