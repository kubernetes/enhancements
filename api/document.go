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

package api

import (
	"io"
)

// TODO(api): Populate interface
// TODO(api): Mock interface
type File interface {
	Parse(io.Reader) (Document, error)
}

// TODO(api): Populate interface
// TODO(api): Mock interface
// Document is an interface satisfied by the following types:
// - `Proposal` (KEP)
// - `PRRApproval`
// - `Receipt` (coming soon)
type Document interface {
	Validate() error
}

type Parser struct {
	Groups       []string
	PRRApprovers []string

	Errors []error
}
