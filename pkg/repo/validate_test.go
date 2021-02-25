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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"k8s.io/enhancements/pkg/repo"
)

const (
	testdataDir = "testdata"
	reposDir    = "repos"
)

var validRepo = filepath.Join(
	testdataDir,
	reposDir,
	"valid",
)

func TestRepoValidate(t *testing.T) {
	testcases := []struct {
		name        string
		repoPath    string
		expectError bool
	}{
		{
			name:        "valid",
			repoPath:    "valid",
			expectError: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			repoPath := filepath.Join(testdataDir, reposDir, tc.repoPath)

			r, repoErr := repo.New(repoPath)
			require.NoError(t, repoErr)

			warnings, valErrMap, err := r.Validate()
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, valErrMap, 0)
			}

			t.Logf(
				"KEP validation succeeded, but the following warnings occurred: %v",
				warnings,
			)
		})
	}
}
