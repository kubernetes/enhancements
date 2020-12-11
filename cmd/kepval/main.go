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
	"io"
	"os"

	"k8s.io/enhancements/pkg/kepval/keps"
)

// TODO: Can we remove this file?

func main() {
	os.Exit(run(os.Stderr))
}

func run(w io.Writer) int {
	parser := &keps.Parser{}
	for _, filename := range os.Args[1:] {
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(w, "could not open file %s: %v\n", filename, err)
			return 1
		}
		defer file.Close()
		kep := parser.Parse(file)
		if kep.Error != nil {
			fmt.Fprintf(w, "%v has an error: %q\n", filename, kep.Error.Error())
			return 1
		}
	}

	fmt.Fprintf(w, "No validation errors: %v\n", os.Args[1:])
	return 0
}
