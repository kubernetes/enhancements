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
	"k8s.io/enhancements/api"
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
			err: fmt.Errorf("no SIGs match any of: [sig-does-not-exist]"),
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

func TestQuery(t *testing.T) {
	testcases := map[string][]struct {
		name      string
		queryOpts repo.QueryOpts
		err       error
		kepNames  []string
		skip      bool
	}{
		"all": {
			{
				name:      "defaults",
				queryOpts: repo.QueryOpts{},
				kepNames: []string{
					"1-the-bare-minimum",
					"123-newstyle",
					"13-keps-as-crds",
					"404-question-not-found",
					"42-the-answer",
				},
			},
			{
				name: "everything",
				queryOpts: repo.QueryOpts{
					Groups: []string{"sig-architecture"},
					Status: []string{string(api.ProvisionalStatus)},
					Stage:  []string{string(api.AlphaStage)},
					// TODO: PRRApprover: []string{""},
					Author:   []string{"@rjbez17"},
					Approver: []string{"@liggitt"},
				},
				kepNames: []string{
					"123-newstyle",
				},
			},
		},
		"groups": {
			{
				name: "results",
				queryOpts: repo.QueryOpts{
					Groups: []string{"sig-owns-participates"},
				},
				kepNames: []string{"1-the-bare-minimum", "13-keps-as-crds"},
			},
			{
				name: "no results",
				queryOpts: repo.QueryOpts{
					Groups: []string{"sig-participates-only"},
				},
			},
			{
				name: "invalid",
				queryOpts: repo.QueryOpts{
					Groups: []string{"sig-does-not-exist"},
				},
				err: fmt.Errorf("invalid query options: no SIGs match any of: [sig-does-not-exist]"),
			},
		},
		"status": {
			{
				name: "results",
				queryOpts: repo.QueryOpts{
					Status: []string{string(api.ProvisionalStatus)},
				},
				kepNames: []string{
					"123-newstyle",
					"13-keps-as-crds",
				},
			},
			{
				name: "TODO: invalid should error, but instead returns nothing",
				queryOpts: repo.QueryOpts{
					Status: []string{"status-does-not-exist"},
				},
				err:  fmt.Errorf("something about invalid status"),
				skip: true,
			},
		},
		"stage": {
			{
				name: "empty-string",
				queryOpts: repo.QueryOpts{
					Stage: []string{""},
				},
				kepNames: []string{
					"1-the-bare-minimum",
				},
			},
			{
				name: "none",
				queryOpts: repo.QueryOpts{
					Stage: []string{"none"},
				},
				kepNames: []string{
					"1-the-bare-minimum",
				},
			},
			{
				name: "alpha",
				queryOpts: repo.QueryOpts{
					Stage: []string{string(api.AlphaStage)},
				},
				kepNames: []string{
					"123-newstyle",
					"42-the-answer",
				},
			},
			{
				name: "beta",
				queryOpts: repo.QueryOpts{
					Stage: []string{string(api.BetaStage)},
				},
				kepNames: []string{
					"404-question-not-found",
				},
			},
			{
				name: "stable",
				queryOpts: repo.QueryOpts{
					Stage: []string{string(api.StableStage)},
				},
				kepNames: []string{
					"13-keps-as-crds",
				},
			},
			{
				name: "TODO: invalid should error, but instead returns nothing",
				queryOpts: repo.QueryOpts{
					Stage: []string{"stage-does-not-exist"},
				},
				err:  fmt.Errorf("something about invalid stage"),
				skip: true,
			},
		},
		"author": {
			{
				name: "results",
				queryOpts: repo.QueryOpts{
					Author: []string{"@alice"},
				},
				kepNames: []string{
					"404-question-not-found",
					"42-the-answer",
					"13-keps-as-crds",
				},
			},
			{
				name: "no results",
				queryOpts: repo.QueryOpts{
					Author: []string{"authors-nothing"},
				},
			},
			{
				name: "TODO: empty-string should error but instead returns nothing",
				queryOpts: repo.QueryOpts{
					Author: []string{""},
				},
				err:  fmt.Errorf("something about invalid author"),
				skip: true,
			},
		},
		"approver": {
			{
				name: "results",
				queryOpts: repo.QueryOpts{
					Approver: []string{"@carolyn"},
				},
				kepNames: []string{
					"404-question-not-found",
					"42-the-answer",
					"13-keps-as-crds",
				},
			},
			{
				name: "no results",
				queryOpts: repo.QueryOpts{
					Approver: []string{"approves-nothing"},
				},
			},
			{
				name: "TODO: empty-string should error but instead returns nothing",
				queryOpts: repo.QueryOpts{
					Author: []string{""},
				},
				err:  fmt.Errorf("something about invalid approver"),
				skip: true,
			},
		},
		"participating-sig": {
			{
				name: "results",
				queryOpts: repo.QueryOpts{
					Participant: []string{"sig-participates-only"},
				},
				kepNames: []string{
					"42-the-answer",
					"404-question-not-found",
				},
			},
			{
				name: "no results",
				queryOpts: repo.QueryOpts{
					Participant: []string{"sig-owns-only"},
				},
			},
			{
				name: "invalid",
				queryOpts: repo.QueryOpts{
					Groups: []string{"sig-does-not-exist"},
				},
				err: fmt.Errorf("invalid query options: no SIGs match any of: [sig-does-not-exist]"),
			},
		},
	}

	r := fixture.validRepo

	for group, groupTestcases := range testcases {
		for _, tc := range groupTestcases {
			t.Run(fmt.Sprintf("%s/%s", group, tc.name), func(t *testing.T) {
				if tc.skip {
					t.Skip("TODO")
				}
				// TODO(spiffxp): uh, Query shouldn't care about Output
				queryOpts := tc.queryOpts
				queryOpts.Output = "json"

				results, err := r.Query(&queryOpts)

				if tc.err == nil {
					require.Nil(t, err)
				} else {
					// TODO(spiffxp): does this verify not nil, or the actual error?
					require.Error(t, err, tc.err.Error())
				}

				if len(tc.kepNames) != len(results) {
					t.Errorf("expected %v results, found: %v", len(tc.kepNames), len(results))
				}

				expectedKEPs := make(map[string]bool)
				for _, name := range tc.kepNames {
					expectedKEPs[name] = false
				}

				errs := []error{}

				for _, kep := range results {
					if _, expected := expectedKEPs[kep.Name]; expected {
						expectedKEPs[kep.Name] = true
					} else {
						errs = append(errs, fmt.Errorf("query returned unexpected kep: %+v", kep))
					}
				}

				for _, name := range tc.kepNames {
					if found := expectedKEPs[name]; !found {
						errs = append(errs, fmt.Errorf("query did not return expected kep: %v", name))
					}
				}

				for _, err := range errs {
					t.Error(err)
				}
			})
		}
	}
}
