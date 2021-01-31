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

package cmd

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"k8s.io/enhancements/pkg/kepctl"
	"k8s.io/release/pkg/log"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:               "kepctl",
	Short:             "kepctl helps you build keps",
	PersistentPreRunE: initLogging,
}

type rootOptions struct {
	logLevel string
}

var rootOpts = &rootOptions{}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&rootOpts.logLevel,
		"log-level",
		"info",
		fmt.Sprintf("the logging verbosity, either %s", log.LevelNames()),
	)
}

func initLogging(*cobra.Command, []string) error {
	return log.SetupGlobalLogger(rootOpts.logLevel)
}

// TODO: Refactor/remove below

func main() {
	cmd, err := buildMainCommand()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildMainCommand() (*cobra.Command, error) {
	repoPath := os.Getenv("ENHANCEMENTS_PATH")
	k, err := kepctl.New(repoPath)
	if err != nil {
		return nil, err
	}

	rootCmd := &cobra.Command{
		Use:   "kepctl",
		Short: "kepctl helps you build keps",
	}

	rootCmd.AddCommand(buildCreateCommand(k))
	rootCmd.AddCommand(buildPromoteCommand(k))
	rootCmd.AddCommand(buildQueryCommand(k))
	return rootCmd, nil
}
