/*
Copyright 2019 The Kubernetes Authors.

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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	kepsDir = "keps"
)

// This is the actual validation check of all keps in this repo
func TestValidation(t *testing.T) {
	// Find all the keps
	files := []string{}
	err := filepath.Walk(
		filepath.Join("..", "..", kepsDir),
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
	// This indicates a problem walking the filepath, not a validation error.
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatal("must find more than 0 keps")
	}

	// Overwrite the command line argument for the run() function
	os.Args = []string{"", ""}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			os.Args[1] = file
			if exit := run(); exit != 0 {
				t.Fatalf("exit code was %d and not 0. Please see output.", exit)
			}
		})
	}
}

// ignore certain files in the keps/ subdirectory
func ignore(name string) bool {
	if !strings.HasSuffix(name, "md") {
		return true
	}
	if name == "0023-documentation-for-images.md" ||
		name == "0004-cloud-provider-template.md" ||
		name == "README.md" ||
		name == "kep-faq.md" {
		return true
	}
	return false
}
