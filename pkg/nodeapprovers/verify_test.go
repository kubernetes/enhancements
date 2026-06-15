/*
Copyright The Kubernetes Authors.

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

package nodeapprovers

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// violationsFor returns the expected violations for a fixture directory, with
// KEPPath filled in to point at that fixture's kep.yaml.
func violationsFor(dir string, partials ...Violation) []Violation {
	kepPath := filepath.Join("testdata", dir, "kep.yaml")
	out := make([]Violation, 0, len(partials))
	for _, p := range partials {
		p.KEPPath = kepPath
		out = append(out, p)
	}
	return out
}

func TestVerifyKEP(t *testing.T) {
	testcases := []struct {
		name string
		dir  string
		want []Violation
	}{
		{
			name: "valid",
			dir:  "valid",
		},
		{
			// Mixed-case handles in kep.yaml must match lowercase OWNERS
			// entries after normalization, so no violations are expected.
			name: "mixed case normalizes",
			dir:  "mixed-case",
		},
		{
			name: "missing reviewer",
			dir:  "missing-reviewer",
			want: violationsFor("missing-reviewer",
				Violation{
					Role:   reviewerRole,
					User:   "someoneelse",
					Reason: "not listed under reviewers in OWNERS",
				},
				Violation{
					Role:   reviewerRole,
					User:   "tallclair",
					Reason: "listed in OWNERS but not annotated as sig-node-assigned-reviewer in kep.yaml",
				},
			),
		},
		{
			name: "missing approver",
			dir:  "missing-approver",
			want: violationsFor("missing-approver",
				Violation{
					Role:   approverRole,
					User:   "someoneelse",
					Reason: "not listed under approvers in OWNERS",
				},
				Violation{
					Role:   approverRole,
					User:   "tallclair",
					Reason: "listed in OWNERS but not annotated as sig-node-assigned-approver in kep.yaml",
				},
			),
		},
		{
			name: "no owners",
			dir:  "no-owners",
			want: violationsFor("no-owners",
				Violation{
					Role:   reviewerRole,
					User:   "tallclair",
					Reason: "OWNERS file not found",
				},
				Violation{
					Role:   approverRole,
					User:   "dchen1107",
					Reason: "OWNERS file not found",
				},
			),
		},
		{
			name: "no markers but OWNERS exists",
			dir:  "no-markers",
			want: violationsFor("no-markers",
				Violation{
					Role:   reviewerRole,
					User:   "tallclair",
					Reason: "listed in OWNERS but not annotated as sig-node-assigned-reviewer in kep.yaml",
				},
				Violation{
					Role:   approverRole,
					User:   "dchen1107",
					Reason: "listed in OWNERS but not annotated as sig-node-assigned-approver in kep.yaml",
				},
			),
		},
		{
			name: "extra in owners",
			dir:  "extra-in-owners",
			want: violationsFor("extra-in-owners",
				Violation{
					Role:   reviewerRole,
					User:   "mrunalp",
					Reason: "listed in OWNERS but not annotated as sig-node-assigned-reviewer in kep.yaml",
				},
				Violation{
					Role:   approverRole,
					User:   "derekwaynecarr",
					Reason: "listed in OWNERS but not annotated as sig-node-assigned-approver in kep.yaml",
				},
			),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			kepPath := filepath.Join("testdata", tc.dir, "kep.yaml")
			violations, err := VerifyKEP(kepPath)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.want, violations, "violations: %v", violations)
		})
	}
}

func TestVerifyAll(t *testing.T) {
	violations, err := VerifyAll("testdata")
	require.NoError(t, err)

	// Aggregated violations across all fixtures: missing-reviewer,
	// missing-approver, no-owners, extra-in-owners, and no-markers each
	// contribute two, and valid and mixed-case contribute none.
	want := make([]Violation, 0, 10)
	want = append(want, violationsFor("missing-reviewer",
		Violation{
			Role:   reviewerRole,
			User:   "someoneelse",
			Reason: "not listed under reviewers in OWNERS",
		},
		Violation{
			Role:   reviewerRole,
			User:   "tallclair",
			Reason: "listed in OWNERS but not annotated as sig-node-assigned-reviewer in kep.yaml",
		},
	)...)
	want = append(want, violationsFor("missing-approver",
		Violation{
			Role:   approverRole,
			User:   "someoneelse",
			Reason: "not listed under approvers in OWNERS",
		},
		Violation{
			Role:   approverRole,
			User:   "tallclair",
			Reason: "listed in OWNERS but not annotated as sig-node-assigned-approver in kep.yaml",
		},
	)...)
	want = append(want, violationsFor("no-owners",
		Violation{
			Role:   reviewerRole,
			User:   "tallclair",
			Reason: "OWNERS file not found",
		},
		Violation{
			Role:   approverRole,
			User:   "dchen1107",
			Reason: "OWNERS file not found",
		},
	)...)
	want = append(want, violationsFor("extra-in-owners",
		Violation{
			Role:   reviewerRole,
			User:   "mrunalp",
			Reason: "listed in OWNERS but not annotated as sig-node-assigned-reviewer in kep.yaml",
		},
		Violation{
			Role:   approverRole,
			User:   "derekwaynecarr",
			Reason: "listed in OWNERS but not annotated as sig-node-assigned-approver in kep.yaml",
		},
	)...)
	want = append(want, violationsFor("no-markers",
		Violation{
			Role:   reviewerRole,
			User:   "tallclair",
			Reason: "listed in OWNERS but not annotated as sig-node-assigned-reviewer in kep.yaml",
		},
		Violation{
			Role:   approverRole,
			User:   "dchen1107",
			Reason: "listed in OWNERS but not annotated as sig-node-assigned-approver in kep.yaml",
		},
	)...)

	require.ElementsMatch(t, want, violations, "violations: %v", violations)
}
