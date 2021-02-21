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
	"testing"

	"github.com/stretchr/testify/require"

	"k8s.io/enhancements/pkg/repo"
)

func TestRepoValidate(t *testing.T) {
	r, repoErr := repo.New("testdata")
	require.Nil(t, repoErr)

	warnings, valErrMap, err := r.Validate()
	require.Nil(t, err)
	require.Len(t, valErrMap, 0)

	t.Logf(
		"KEP validation succeeded, but the following warnings occurred: %v",
		warnings,
	)
}
