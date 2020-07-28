package main

import (
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
	f.BoolVar(&opts.IncludePRs, "include-prs", false, "Include PRs in the results")

	addRepoPathFlag(f, &opts.CommonArgs)

	return cmd
}
