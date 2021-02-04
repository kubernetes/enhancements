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
	"github.com/spf13/cobra"
	"k8s.io/enhancements/pkg/kepctl"
)

func buildPromoteCommand(k *kepctl.Client) *cobra.Command {
	opts := kepctl.PromoteOpts{}
	cmd := &cobra.Command{
		Use:     "promote [KEP]",
		Short:   "Promote a KEP",
		Long:    "Promote a KEP to a new stage for a target release",
		Example: `  kepctl promote sig-architecture/000-mykep --stage beta --release v1.20`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.Validate(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return k.Promote(opts)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&opts.Stage, "stage", "s", "", "KEP Stage")
	f.StringVarP(&opts.Release, "release", "r", "", "Target Release")

	addRepoPathFlag(f, &opts.CommonArgs)

	return cmd
}
