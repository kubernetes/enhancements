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
	flag "github.com/spf13/pflag"

	"k8s.io/enhancements/pkg/kepctl"
)

func addRepoPathFlag(
	f *flag.FlagSet,
	opts *kepctl.CommonArgs,
) {
	f.StringVar(&opts.RepoPath, "repo-path", "", "Path to kubernetes/enhancements")
	f.StringVar(&opts.TokenPath, "gh-token-path", "", "Path to a file with a GitHub API token")
}
