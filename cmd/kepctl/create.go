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

func buildCreateCommand(k *kepctl.Client) *cobra.Command {

	opts := kepctl.CreateOpts{}
	cmd := &cobra.Command{
		Use:     "create [KEP]",
		Short:   "Create a new KEP",
		Long:    "Create a new KEP using the current KEP template for the given type",
		Example: `  kepctl create sig-architecture/000-mykep`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.Validate(args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return k.Create(opts)
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.Title, "title", "", "KEP Title")
	f.StringArrayVar(&opts.Authors, "authors", []string{}, "Authors")
	f.StringArrayVar(&opts.Reviewers, "reviewers", []string{}, "Reviewers")
	f.StringVar(&opts.Type, "type", "feature", "KEP Type")
	f.StringVarP(&opts.State, "state", "s", "provisional", "KEP State")
	f.StringArrayVar(&opts.SIGS, "sigs", []string{}, "Participating SIGs")
	f.StringArrayVar(&opts.PRRApprovers, "prr-approver", []string{}, "PRR Approver")

	addRepoPathFlag(f, &opts.CommonArgs)
	return cmd
}
