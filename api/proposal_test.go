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

package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/enhancements/api"
)

func TestProposal_IsMissingMilestone(t *testing.T) {
	testcases := []struct {
		name     string
		proposal *api.Proposal
		expected bool
	}{
		{
			name: "milestone is present",
			proposal: &api.Proposal{
				LatestMilestone: "v1.36",
			},
			expected: false,
		},
		{
			name: "milestone is empty",
			proposal: &api.Proposal{
				LatestMilestone: "",
			},
			expected: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.proposal.IsMissingMilestone())
		})
	}
}

func TestProposal_IsMissingStage(t *testing.T) {
	testcases := []struct {
		name     string
		proposal *api.Proposal
		expected bool
	}{
		{
			name: "stage is alpha",
			proposal: &api.Proposal{
				Stage: api.AlphaStage,
			},
			expected: false,
		},
		{
			name: "stage is beta",
			proposal: &api.Proposal{
				Stage: api.BetaStage,
			},
			expected: false,
		},
		{
			name: "stage is stable",
			proposal: &api.Proposal{
				Stage: api.StableStage,
			},
			expected: false,
		},
		{
			name: "stage is empty",
			proposal: &api.Proposal{
				Stage: "",
			},
			expected: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.proposal.IsMissingStage())
		})
	}
}

func TestKEPHandler_Validate(t *testing.T) {
	parser := api.KEPHandler{}
	parser.Groups = []string{"sig-architecture", "sig-node", "sig-network"}
	parser.PRRApprovers = []string{"@wojtek-t"}

	testcases := []struct {
		name     string
		proposal *api.Proposal
		wantErrs int
	}{
		{
			name: "valid proposal",
			proposal: &api.Proposal{
				Title:      "Test KEP",
				Number:     "1234",
				Authors:    []string{"@alice"},
				OwningSIG:  "sig-architecture",
				Approvers:  []string{"@bob"},
				Status:     api.ProvisionalStatus,
				Stage:      api.AlphaStage,
			},
			wantErrs: 0,
		},
		{
			name: "invalid owning-sig",
			proposal: &api.Proposal{
				Title:      "Test KEP",
				Number:     "1234",
				Authors:    []string{"@alice"},
				OwningSIG:  "sig-nonexistent",
				Approvers:  []string{"@bob"},
				Status:     api.ProvisionalStatus,
				Stage:      api.AlphaStage,
			},
			wantErrs: 1,
		},
		{
			name: "invalid status",
			proposal: &api.Proposal{
				Title:      "Test KEP",
				Number:     "1234",
				Authors:    []string{"@alice"},
				OwningSIG:  "sig-architecture",
				Approvers:  []string{"@bob"},
				Status:     "invalid-status",
				Stage:      api.AlphaStage,
			},
			wantErrs: 1,
		},
		{
			name: "invalid stage",
			proposal: &api.Proposal{
				Title:      "Test KEP",
				Number:     "1234",
				Authors:    []string{"@alice"},
				OwningSIG:  "sig-architecture",
				Approvers:  []string{"@bob"},
				Status:     api.ProvisionalStatus,
				Stage:      "invalid-stage",
			},
			wantErrs: 1,
		},
		{
			name: "implemented status with non-stable stage",
			proposal: &api.Proposal{
				Title:      "Test KEP",
				Number:     "1234",
				Authors:    []string{"@alice"},
				OwningSIG:  "sig-architecture",
				Approvers:  []string{"@bob"},
				Status:     api.ImplementedStatus,
				Stage:      api.BetaStage,
			},
			wantErrs: 1,
		},
		{
			name: "multiple errors: invalid sig and invalid status",
			proposal: &api.Proposal{
				Title:      "Test KEP",
				Number:     "1234",
				Authors:    []string{"@alice"},
				OwningSIG:  "sig-nonexistent",
				Approvers:  []string{"@bob"},
				Status:     "invalid-status",
				Stage:      api.AlphaStage,
			},
			wantErrs: 2,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			errs := parser.Validate(tc.proposal)
			require.Len(t, errs, tc.wantErrs)
		})
	}
}

func TestStage_IsValid(t *testing.T) {
	testcases := []struct {
		name    string
		stage   api.Stage
		wantErr bool
	}{
		{name: "alpha", stage: api.AlphaStage, wantErr: false},
		{name: "beta", stage: api.BetaStage, wantErr: false},
		{name: "stable", stage: api.StableStage, wantErr: false},
		{name: "deprecated", stage: api.Deprecated, wantErr: false},
		{name: "disabled", stage: api.Disabled, wantErr: false},
		{name: "removed", stage: api.Removed, wantErr: false},
		{name: "invalid", stage: "invalid", wantErr: true},
		{name: "empty", stage: "", wantErr: true},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.stage.IsValid()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	testcases := []struct {
		name    string
		status  api.Status
		wantErr bool
	}{
		{name: "provisional", status: api.ProvisionalStatus, wantErr: false},
		{name: "implementable", status: api.ImplementableStatus, wantErr: false},
		{name: "implemented", status: api.ImplementedStatus, wantErr: false},
		{name: "deferred", status: api.DeferredStatus, wantErr: false},
		{name: "rejected", status: api.RejectedStatus, wantErr: false},
		{name: "withdrawn", status: api.WithdrawnStatus, wantErr: false},
		{name: "replaced", status: api.ReplacedStatus, wantErr: false},
		{name: "invalid", status: "invalid", wantErr: true},
		{name: "empty", status: "", wantErr: true},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.status.IsValid()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
