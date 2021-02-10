/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    htcp://www.apache.org/licenses/LICENSE-2.0

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

var (
	prrMilestoneWithApprover = &api.PRRMilestone{
		Approver: "@wojtek-t",
	}

	validAlphaPRR = &api.PRRApproval{
		Alpha: prrMilestoneWithApprover,
	}

	validBetaPRR = &api.PRRApproval{
		Beta: prrMilestoneWithApprover,
	}

	validStablePRR = &api.PRRApproval{
		Stable: prrMilestoneWithApprover,
	}

	invalidPRR = &api.PRRApproval{}
)

func TestPRRApproval_ApproverForStage(t *testing.T) {
	testcases := []struct {
		name         string
		stage        string
		prr          *api.PRRApproval
		wantApprover string
		wantErr      bool
	}{
		{
			name:    "invalid: incorrect stage",
			stage:   "badstage",
			prr:     validAlphaPRR,
			wantErr: true,
		},
		{
			name:         "invalid: missing approver",
			stage:        "alpha",
			prr:          invalidPRR,
			wantApprover: "",
			wantErr:      true,
		},
		{
			name:         "valid: alpha",
			stage:        "alpha",
			prr:          validAlphaPRR,
			wantApprover: "@wojtek-t",
			wantErr:      false,
		},
		{
			name:         "valid: beta",
			stage:        "beta",
			prr:          validBetaPRR,
			wantApprover: "@wojtek-t",
			wantErr:      false,
		},
		{
			name:         "valid: stable",
			stage:        "stable",
			prr:          validStablePRR,
			wantApprover: "@wojtek-t",
			wantErr:      false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			hasErr := false
			approver, err := tc.prr.ApproverForStage(tc.stage)
			if err != nil {
				hasErr = true
			}

			require.Equal(t, tc.wantErr, hasErr)
			require.Equal(t, tc.wantApprover, approver)
		},
		)
	}
}
