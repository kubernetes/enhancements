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

package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"k8s.io/enhancements/pkg/kepval"
)

const (
	kepsDir = "keps"
)

func TestValidation(t *testing.T) {
	wd, err := os.Getwd()
	require.Nil(t, err)

	rootDir := filepath.Dir(wd)
	kepDir := filepath.Join(rootDir, kepsDir)
	prrDir := filepath.Join(kepDir, kepval.DefaultPRRDir)

	repo := kepval.NewRepoClient()
	repo.SetOptions(
		repo.WithKEPDir(kepDir),
		repo.WithPRRDir(prrDir),
	)

	//require.NotNil(t, repo.)

	warnings, valErrMap, err := repo.Validate()
	require.Nil(t, err)
	require.Len(t, valErrMap, 0)

	t.Logf(
		"KEP validation succeeded, but the following warnings occurred: %v",
		warnings,
	)
}
