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

package repo_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/repo"
	"sigs.k8s.io/yaml"
)

// TODO: Consider using afero to mock the filesystem here
func TestWriteKep(t *testing.T) {
	testcases := []struct {
		name         string
		kepFile      string
		repoPath     string
		kepName      string
		sig          string
		expectedPath string
		expectError  bool
	}{
		{
			name:         "simple KEP",
			kepFile:      "testdata/valid-kep.yaml",
			repoPath:     "enhancements",
			kepName:      "1010-test",
			sig:          "sig-auth",
			expectedPath: filepath.Join("enhancements", "keps", "sig-auth", "1010-test"),
			expectError:  false,
		},
		{
			name:         "missing KEP name",
			kepFile:      "testdata/valid-kep.yaml",
			repoPath:     "enhancements",
			sig:          "sig-auth",
			expectedPath: filepath.Join("enhancements", "keps", "sig-auth", "1010-test"),
			expectError:  true,
		},
		{
			name:         "missing owning SIG",
			kepFile:      "testdata/valid-kep.yaml",
			repoPath:     "enhancements",
			kepName:      "1010-test",
			expectedPath: filepath.Join("enhancements", "keps", "sig-auth", "1010-test"),
			expectError:  true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := ioutil.TempDir("", "")
			mkErr := os.MkdirAll(
				filepath.Join(
					tempDir,
					tc.repoPath,
					repo.ProposalPathStub,
					repo.PRRApprovalPathStub,
				),
				os.ModePerm,
			)

			require.Nil(t, mkErr)

			templatePath := filepath.Join(
				tempDir,
				tc.repoPath,
				repo.ProposalPathStub,
				repo.ProposalTemplatePathStub,
			)

			mkErr = os.MkdirAll(
				templatePath,
				os.ModePerm,
			)

			require.Nil(t, mkErr)

			templateFile := filepath.Join(templatePath, repo.ProposalFilename)
			emptyTemplate, fileErr := os.Create(templateFile)
			require.Nil(t, fileErr)
			emptyTemplate.Close()

			defer func() {
				t.Logf("cleanup!")
				err := os.RemoveAll(tempDir)
				if err != nil {
					t.Logf("error cleaning up test: %s", err)
				}
			}()

			require.NoError(t, err)

			repoPath := tc.repoPath
			repoPath = filepath.Join(tempDir, repoPath)

			proposalReadme := filepath.Join(repoPath, repo.ProposalPathStub, "README.md")
			emptyReadme, fileErr := os.Create(proposalReadme)
			require.Nil(t, fileErr)
			emptyReadme.Close()

			c := newTestClient(t, repoPath)

			b, err := ioutil.ReadFile(tc.kepFile)
			require.NoError(t, err)

			var p api.Proposal
			err = yaml.Unmarshal(b, &p)
			require.NoError(t, err)

			p.OwningSIG = tc.sig
			p.Name = tc.kepName

			err = c.r.WriteKEP(&p)

			files, readErr := ioutil.ReadDir(c.r.ProposalPath)
			require.Nil(t, readErr)

			for _, f := range files {
				t.Logf(f.Name())
			}

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				computedPath := filepath.Join(tempDir, tc.expectedPath)
				dirStat, err := os.Stat(computedPath)
				require.NoError(t, err)
				require.NotNil(t, dirStat)
				require.True(t, dirStat.IsDir())
				p := filepath.Join(computedPath, repo.ProposalMetadataFilename)
				fileStat, err := os.Stat(p)
				require.NoError(t, err)
				require.NotNil(t, fileStat)
			}
		})
	}
}

type testClient struct {
	T *testing.T
	b *bytes.Buffer
	r *repo.Repo
}

func newTestClient(t *testing.T, repoPath string) testClient {
	b := &bytes.Buffer{}
	tc := testClient{
		T: t,
		b: b,
	}
	fetcher := &api.MockGroupFetcher{
		Groups: []string{"sig-auth", "sig-api-machinery", "sig-architecture"},
	}
	r, err := repo.NewRepo(repoPath, fetcher)
	require.Nil(t, err)

	tc.r = r

	// TODO: Parameterize
	tc.addTemplate(repo.ProposalMetadataFilename)
	tc.addTemplate(repo.ProposalFilename)

	return tc
}

func (tc *testClient) addTemplate(file string) {
	src := filepath.Join(
		fixture.validRepoPath,
		repo.ProposalPathStub,
		repo.ProposalTemplatePathStub,
		file,
	)

	data, err := ioutil.ReadFile(src)
	if err != nil {
		tc.T.Fatal(err)
	}

	dirPath := filepath.Join(tc.r.BasePath, repo.ProposalPathStub, repo.ProposalTemplatePathStub)
	err = os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		tc.T.Fatal(err)
	}

	dest := filepath.Join(dirPath, file)
	tc.T.Logf("Writing %s to %s", file, dest)
	err = ioutil.WriteFile(dest, data, os.ModePerm)
	if err != nil {
		tc.T.Fatal(err)
	}
}
