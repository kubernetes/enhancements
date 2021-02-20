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

package proposal

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"k8s.io/enhancements/api"
	"sigs.k8s.io/yaml"
)

func TestWriteKep(t *testing.T) {
	testcases := []struct {
		name         string
		kepFile      string
		repoPath     string
		opts         CreateOpts
		expectedPath string
		expectError  bool
	}{
		{
			name:     "simple kep",
			kepFile:  "testdata/valid-kep.yaml",
			repoPath: "enhancements",
			opts: CreateOpts{
				CommonArgs: CommonArgs{
					Name: "1010-test",
					SIG:  "sig-auth",
				},
			},
			expectedPath: filepath.Join("enhancements", "keps", "sig-auth", "1010-test"),
			expectError:  false,
		},
		{
			name:     "opts repo path works",
			kepFile:  "testdata/valid-kep.yaml",
			repoPath: "",
			opts: CreateOpts{
				CommonArgs: CommonArgs{
					Name:     "1011-test",
					SIG:      "sig-architecture",
					RepoPath: "enhancementz",
				},
			},
			expectedPath: filepath.Join("enhancementz", "keps", "sig-architecture", "1011-test"),
			expectError:  false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := ioutil.TempDir("", "")
			defer func() {
				t.Logf("cleanup!")
				err := os.RemoveAll(tempDir)
				if err != nil {
					t.Logf("error cleaning up test: %s", err)
				}
			}()
			require.NoError(t, err)
			repoPath := tc.repoPath
			if repoPath == "" {
				repoPath = tc.opts.RepoPath
			}

			repoPath = filepath.Join(tempDir, repoPath)
			c := newTestClient(t, repoPath)

			b, err := ioutil.ReadFile(tc.kepFile)
			require.NoError(t, err)

			var p api.Proposal
			err = yaml.Unmarshal(b, &p)
			require.NoError(t, err)

			tc.opts.CommonArgs.RepoPath = repoPath
			err = c.writeKEP(&p, &tc.opts.CommonArgs)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				computedPath := filepath.Join(tempDir, tc.expectedPath)
				dirStat, err := os.Stat(computedPath)
				require.NoError(t, err)
				require.NotNil(t, dirStat)
				require.True(t, dirStat.IsDir())
				p := filepath.Join(computedPath, "kep.yaml")
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
	*Client
}

func newTestClient(t *testing.T, repoPath string) testClient {
	b := &bytes.Buffer{}
	tc := testClient{
		T: t,
		b: b,
		Client: &Client{
			RepoPath: repoPath,
			Out:      b,
		},
	}

	// TODO: Parameterize
	tc.addTemplate("kep.yaml")
	tc.addTemplate("README.md")
	return tc
}

func (tc *testClient) addTemplate(file string) {
	src := filepath.Join("testdata", "templates", file)
	data, err := ioutil.ReadFile(src)
	if err != nil {
		tc.T.Fatal(err)
	}

	dirPath := filepath.Join(tc.Client.RepoPath, "keps", "NNNN-kep-template")
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
