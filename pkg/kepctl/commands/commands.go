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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"k8s.io/enhancements/pkg/repo"
	"k8s.io/release/pkg/log"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:               "kepctl",
	Short:             "kepctl helps you build KEPs",
	PersistentPreRunE: initLogging,
}

var rootOpts = &repo.Options{}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&rootOpts.LogLevel,
		"log-level",
		"info",
		fmt.Sprintf("the logging verbosity, either %s", log.LevelNames()),
	)

	// TODO: This should be defaulted in the package instead
	rootCmd.PersistentFlags().StringVar(
		&rootOpts.RepoPath,
		"repo-path",
		os.Getenv("ENHANCEMENTS_PATH"),
		"path to kubernetes/enhancements",
	)

	rootCmd.PersistentFlags().StringVar(
		&rootOpts.TokenPath,
		"gh-token-path",
		"",
		"path to a file with a GitHub API token",
	)
}

func initLogging(*cobra.Command, []string) error {
	return log.SetupGlobalLogger(rootOpts.LogLevel)
}
