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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/legacy/keps"
)

func Usage() {
	fmt.Fprintf(os.Stderr, `
Usage: %s [-dir <kep-directory>] [-output <path-to-json-file>]
Command line flags override config values.
`, os.Args[0])
	flag.PrintDefaults()
}

func main() {
	dirPath := flag.String("dir", "keps", "root directory for the KEPs")
	filePath := flag.String("output", "keps.json", "output json file")

	flag.Usage = Usage
	flag.Parse()

	if *dirPath == "" {
		fmt.Fprintf(os.Stderr, "please specify the root directory for KEPs using '--dir'\n")
		os.Exit(1)
	}
	if _, err := os.Stat(*dirPath); os.IsNotExist(err) {
		fmt.Printf("directory does not exist : %s", *dirPath)
		os.Exit(1)
	}

	if *filePath == "" {
		fmt.Fprintf(os.Stderr, "please specify the file path for the output json using '--output'\n")
		os.Exit(1)
	}

	// Find all of the KEPs
	files, err := findMarkdownFiles(dirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to find markdown files: %v\n", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "did not find any KEPs\n")
		os.Exit(1)
	}

	// Parse the files
	proposals, err := parseFiles(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing files: %q\n", err)
		os.Exit(1)
	}

	// Generate the json output
	err = printJSONOutput(*filePath, proposals)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not open file: %v\n", err)
		os.Exit(1)
	}
}

func findMarkdownFiles(dirPath *string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(
		*dirPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if ignore(info.Name()) {
				return nil
			}
			files = append(files, path)
			return nil
		},
	)
	return files, err
}

func parseFiles(files []string) (api.Proposals, error) {
	var proposals api.Proposals
	for _, filename := range files {
		parser := &keps.Parser{}
		file, err := os.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("could not open file: %v", err)
		}

		defer file.Close()
		kep := parser.Parse(file)
		// if error is nil we can move on
		if kep.Error != nil {
			return nil, fmt.Errorf("%v has an error: %q", filename, kep.Error.Error())
		}

		fmt.Printf(">>>> parsed file successfully: %s\n", filename)
		proposals.AddProposal(kep)
	}
	return proposals, nil
}

func printJSONOutput(filePath string, proposals api.Proposals) error {
	fmt.Printf("Output file: %s\n", filePath)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	total := len(proposals)
	fmt.Printf("Total KEPs: %d\n", total)

	data, err := json.Marshal(proposals)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filePath, data, 0755); err != nil {
		return err
	}

	return nil
}

// TODO: Consider replacing with a .kepignore file
// ignore certain files in the keps/ subdirectory
func ignore(name string) bool {
	if !strings.HasSuffix(name, "md") {
		return true
	}

	if name == "0023-documentation-for-images.md" ||
		name == "0004-cloud-provider-template.md" ||
		name == "0001a-meta-kep-implementation.md" ||
		name == "0001-kubernetes-enhancement-proposal-process.md" ||
		name == "YYYYMMDD-kep-template.md" ||
		name == "README.md" ||
		name == "kep-faq.md" {
		return true
	}

	return false
}
