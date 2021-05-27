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

package repo_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/enhancements/pkg/repo"
)

func TestValidateQueryOpt(t *testing.T) {
	testcases := []struct {
		name      string
		queryOpts repo.QueryOpts
		err       error
	}{
		{
			name: "groups: valid SIG",
			queryOpts: repo.QueryOpts{
				Groups:     []string{"sig-architecture"},
				IncludePRs: true,
				Output:     "json",
			},
			err: nil,
		},
		{
			name: "groups: invalid SIG",
			queryOpts: repo.QueryOpts{
				Groups:     []string{"sig-does-not-exist"},
				IncludePRs: true,
				Output:     "json",
			},
			err: fmt.Errorf("No SIG matches any of the passed regular expressions"),
		},
		{
			name: "output: unsupported format",
			queryOpts: repo.QueryOpts{
				Groups:     []string{"sig-does-nothing"},
				IncludePRs: true,
				Output:     "PDF",
			},
			err: fmt.Errorf("unsupported output format: PDF. Valid values: [table json yaml]"),
		},
	}

	r := fixture.validRepo

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			queryOpts := tc.queryOpts
			err := r.PrepareQueryOpts(&queryOpts)

			if tc.err == nil {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err, tc.err.Error())
			}
		})
	}
}
