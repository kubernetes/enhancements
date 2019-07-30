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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func TestIntegration(t *testing.T) {
	tempDir, err := ioutil.TempDir(".", "test")
	if err != nil {
		t.Fatalf("%+v", errors.WithStack(err))
	}
	defer os.RemoveAll(tempDir)
	fmt.Println("Cloning...")
	cmd := exec.Command("git", "clone", "https://github.com/kubernetes/enhancements")
	cmd.Dir = tempDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		t.Fatalf("%+v", errors.WithStack(err))
	}
	fmt.Println("Building...")
	cmd = exec.Command("go", "build", "-o", filepath.Join(tempDir, "kepval"), ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(string(out))
		t.Fatal(err)
	}
	fmt.Println("Walking...")
	if filepath.Walk(
		filepath.Join(tempDir, "enhancements", "keps"),
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if err != nil {
				t.Fatalf("%+v", err)
			}
			if ignore(info.Name()) {
				return nil
			}
			cmd = exec.Command(filepath.Join(tempDir, "kepval"), path)
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Println(string(out))
				t.Fatal(err)
			}
			return nil
		},
	) != nil {
		t.Fatal(err)
	}
}

func ignore(name string) bool {
	if !strings.HasSuffix(name, "md") {
		return true
	}
	if name == "0023-documentation-for-images.md" {
		return true
	}
	return false
}
