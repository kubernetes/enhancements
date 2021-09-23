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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/release/pkg/log"

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/repo"
	"k8s.io/enhancements/pkg/yaml"
)

var fixture = struct {
	validRepoPath string
	validRepo     *repo.Repo
}{}

func TestMain(m *testing.M) {
	err := log.SetupGlobalLogger("info")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to setup global logger: %v: ", err)
	}

	fixture.validRepoPath = filepath.Join("testdata", "repos", "valid")

	// NB: keep this in sync with testdata/repos/valid/keps
	fetcher := &api.MockGroupFetcher{
		Groups: []string{
			"sig-architecture",
			"sig-does-nothing",
			"sig-owns-only",
			"sig-owns-participates",
			"sig-participates-only",
		},
	}
	r, err := repo.NewRepo(fixture.validRepoPath, fetcher)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to load repo from %s: %v: ", fixture.validRepoPath, err)
		os.Exit(1)
	}
	fixture.validRepo = r

	os.Exit(m.Run())
}

func TestProposalValidate(t *testing.T) {
	testcases := []struct {
		name string
		file string
		errs []error
	}{
		{
			name: "valid KEP: all fields",
			file: "testdata/valid-kep-full.yaml",
			errs: nil,
		},
		{
			name: "invalid KEP: owning-sig does not exist",
			file: "testdata/invalid-owning-sig.yaml",
			errs: []error{
				fmt.Errorf("invalid owning-sig: not-a-sig"),
			},
		},
		{
			name: "invalid KEP: prr-approver does not exist",
			file: "testdata/invalid-prr-approver.yaml",
			errs: []error{
				fmt.Errorf("invalid prr-approver: not-a-prr-approver"),
			},
		},
	}

	parser := api.KEPHandler{}
	parser.Groups = []string{"sig-api-machinery", "sig-architecture", "sig-auth"}
	parser.PRRApprovers = []string{"@wojtek-t"}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := ioutil.ReadFile(tc.file)
			require.NoError(t, err)

			var p api.Proposal
			err = yaml.UnmarshalStrict(b, &p)
			require.NoError(t, err)

			errs := parser.Validate(&p)
			require.ElementsMatch(t, tc.errs, errs)
		})
	}
}

func TestFindLocalKEPs(t *testing.T) {
	testcases := []struct {
		sig  string
		keps []string
	}{
		{
			"sig-architecture",
			[]string{
				"123-newstyle",
			},
		},
		{
			"sig-sig",
			[]string{},
		},
	}

	r := fixture.validRepo

	for i, tc := range testcases {
		k, err := r.LoadLocalKEPs(tc.sig)
		require.Nil(t, err)

		if len(k) != len(tc.keps) {
			t.Errorf("Test case %d: expected %d but got %d", i, len(tc.keps), len(k))
			continue
		}

		for j, kn := range k {
			if kn.Name != tc.keps[j] {
				t.Errorf("Test case %d: expected %s but got %s", i, tc.keps[j], kn.Name)
			}
		}
	}
}
