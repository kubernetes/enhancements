/*
Copyright 2021 The Kubernetes Authors.

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
	"os"

	"github.com/spf13/cobra"

	"k8s.io/enhancements/pkg/kepctl"
	"k8s.io/release/pkg/log"
)

var rootOpts = &kepctl.Options{}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "kepctl",
		Short:             "kepctl helps you build KEPs",
		PersistentPreRunE: initLogging,
	}

	cmd.PersistentFlags().StringVar(
		&rootOpts.LogLevel,
		"log-level",
		"info",
		fmt.Sprintf("the logging verbosity, either %s", log.LevelNames()),
	)

	// TODO: This should be defaulted in the package instead
	cmd.PersistentFlags().StringVar(
		&rootOpts.RepoPath,
		"repo-path",
		os.Getenv("ENHANCEMENTS_PATH"),
		"path to kubernetes/enhancements",
	)

	cmd.PersistentFlags().StringVar(
		&rootOpts.TokenPath,
		"gh-token-path",
		"",
		"path to a file with a GitHub API token",
	)

	AddCommands(cmd)
	return cmd
}

func AddCommands(topLevel *cobra.Command) {
	addCreate(topLevel)
	addPromote(topLevel)
	addQuery(topLevel)
}

func initLogging(*cobra.Command, []string) error {
	return log.SetupGlobalLogger(rootOpts.LogLevel)
}
