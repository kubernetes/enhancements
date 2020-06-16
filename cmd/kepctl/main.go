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
	"os"

	"github.com/spf13/cobra"
	"k8s.io/enhancements/pkg/kepctl"
)

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
	var rootCmd = &cobra.Command{
		Use:   "kepctl",
		Short: "kepctl helps you build keps",
	}

	rootCmd.AddCommand(buildCreateCommand(k))
	rootCmd.AddCommand(buildPromoteCommand(k))
	return rootCmd, nil
}
