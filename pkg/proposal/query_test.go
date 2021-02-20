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

package proposal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateQueryOpt(t *testing.T) {
	testcases := []struct {
		name      string
		queryOpts QueryOpts
		err       error
	}{
		{
			name: "Valid SIG",
			queryOpts: QueryOpts{
				CommonArgs: CommonArgs{
					Name: "1011-test",
				},
				SIG:        []string{"sig-multicluster"},
				IncludePRs: true,
				Output:     "json",
			},
			err: nil,
		},
		{
			name: "Invalid SIG",
			queryOpts: QueryOpts{
				CommonArgs: CommonArgs{
					Name: "1011-test-xyz",
				},
				SIG:        []string{"sig-xyz"},
				IncludePRs: true,
				Output:     "json",
			},
			err: fmt.Errorf("No SIG matches any of the passed regular expressions"),
		},
		{
			name: "Unsupported Output format",
			queryOpts: QueryOpts{
				CommonArgs: CommonArgs{
					Name: "1011-test-testing",
				},
				SIG:        []string{"sig-testing"},
				IncludePRs: true,
				Output:     "PDF",
			},
			err: fmt.Errorf("unsupported output format: PDF. Valid values: [table json yaml]"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			queryOpts := tc.queryOpts
			err := queryOpts.Validate()

			if tc.err == nil {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err, tc.err.Error())
			}
		})
	}
}
