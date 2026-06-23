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
	"os"
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

// techLeadsFixture is the set of sig-node-tech-leads members defined in
// testdata/techleads/OWNERS_ALIASES.
var techLeadsFixture = map[string]bool{
	"dchen1107": true,
	"mrunalp":   true,
}

// techLeadViolationsFor returns the expected violations for a fixture directory
// under testdata/techleads, with KEPPath filled in.
func techLeadViolationsFor(dir string, partials ...Violation) []Violation {
	kepPath := filepath.Join("testdata", "techleads", dir, "kep.yaml")
	out := make([]Violation, 0, len(partials))
	for _, p := range partials {
		p.KEPPath = kepPath
		out = append(out, p)
	}
	return out
}

func TestVerifyTechLeadApprovers(t *testing.T) {
	testcases := []struct {
		name string
		dir  string
		want []Violation
	}{
		{
			name: "alpha with tech lead is valid",
			dir:  "alpha-valid",
		},
		{
			name: "alpha missing tech lead",
			dir:  "alpha-missing-techlead",
			want: techLeadViolationsFor("alpha-missing-techlead",
				Violation{
					Role:   approverRole,
					Reason: "alpha-stage KEP must list at least one sig-node-tech-leads member as approver",
				},
			),
		},
		{
			name: "alpha with disallowed marker",
			dir:  "alpha-marker-not-allowed",
			want: techLeadViolationsFor("alpha-marker-not-allowed",
				Violation{
					Role:   approverRole,
					User:   "tallclair",
					Reason: "alpha-stage KEP must not use # sig-node-assigned-approver marker",
				},
			),
		},
		{
			name: "alpha listing the sig-node-tech-leads alias is not valid",
			dir:  "alpha-alias-invalid",
			want: techLeadViolationsFor("alpha-alias-invalid",
				Violation{
					Role:   approverRole,
					Reason: "alpha-stage KEP must list at least one sig-node-tech-leads member as approver",
				},
			),
		},
		{
			name: "beta listing the sig-node-tech-leads alias is not valid",
			dir:  "beta-alias-invalid",
			want: techLeadViolationsFor("beta-alias-invalid",
				Violation{
					Role:   approverRole,
					Reason: "non-alpha KEP must list a sig-node-tech-leads member or an approver marked # sig-node-assigned-approver",
				},
			),
		},
		{
			name: "beta with tech lead is valid",
			dir:  "beta-techlead-valid",
		},
		{
			name: "beta with marked approver is valid",
			dir:  "beta-marker-valid",
		},
		{
			name: "beta with neither",
			dir:  "beta-missing",
			want: techLeadViolationsFor("beta-missing",
				Violation{
					Role:   approverRole,
					Reason: "non-alpha KEP must list a sig-node-tech-leads member or an approver marked # sig-node-assigned-approver",
				},
			),
		},
		{
			name: "missing stage is treated as non-alpha",
			dir:  "no-stage",
			want: techLeadViolationsFor("no-stage",
				Violation{
					Role:   approverRole,
					Reason: "non-alpha KEP must list a sig-node-tech-leads member or an approver marked # sig-node-assigned-approver",
				},
			),
		},
		{
			name: "no approvers",
			dir:  "no-approvers",
			want: techLeadViolationsFor("no-approvers",
				Violation{
					Role:   approverRole,
					Reason: "no approvers listed",
				},
			),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			kepPath := filepath.Join("testdata", "techleads", tc.dir, "kep.yaml")
			violations, err := VerifyTechLeadApprovers(kepPath, techLeadsFixture, 37)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.want, violations, "violations: %v", violations)
		})
	}
}

func TestVerifyAllTechLeadApprovers(t *testing.T) {
	root := filepath.Join("testdata", "techleads")
	violations, err := VerifyAllTechLeadApprovers(root, filepath.Join(root, "OWNERS_ALIASES"), 37)
	require.NoError(t, err)

	want := make([]Violation, 0, 7)
	want = append(want, techLeadViolationsFor("alpha-alias-invalid",
		Violation{
			Role:   approverRole,
			Reason: "alpha-stage KEP must list at least one sig-node-tech-leads member as approver",
		},
	)...)
	want = append(want, techLeadViolationsFor("alpha-missing-techlead",
		Violation{
			Role:   approverRole,
			Reason: "alpha-stage KEP must list at least one sig-node-tech-leads member as approver",
		},
	)...)
	want = append(want, techLeadViolationsFor("alpha-marker-not-allowed",
		Violation{
			Role:   approverRole,
			User:   "tallclair",
			Reason: "alpha-stage KEP must not use # sig-node-assigned-approver marker",
		},
	)...)
	want = append(want, techLeadViolationsFor("beta-alias-invalid",
		Violation{
			Role:   approverRole,
			Reason: "non-alpha KEP must list a sig-node-tech-leads member or an approver marked # sig-node-assigned-approver",
		},
	)...)
	want = append(want, techLeadViolationsFor("beta-missing",
		Violation{
			Role:   approverRole,
			Reason: "non-alpha KEP must list a sig-node-tech-leads member or an approver marked # sig-node-assigned-approver",
		},
	)...)
	want = append(want, techLeadViolationsFor("no-stage",
		Violation{
			Role:   approverRole,
			Reason: "non-alpha KEP must list a sig-node-tech-leads member or an approver marked # sig-node-assigned-approver",
		},
	)...)
	want = append(want, techLeadViolationsFor("no-approvers",
		Violation{
			Role:   approverRole,
			Reason: "no approvers listed",
		},
	)...)

	require.ElementsMatch(t, want, violations, "violations: %v", violations)
}

func TestLoadTechLeads(t *testing.T) {
	t.Run("valid file loads the alias members and stays in sync with the fixture", func(t *testing.T) {
		got, err := loadTechLeads(filepath.Join("testdata", "techleads", "OWNERS_ALIASES"))
		require.NoError(t, err)
		// Guards against the hardcoded techLeadsFixture silently drifting from
		// the OWNERS_ALIASES fixture the integration-style test loads.
		require.Equal(t, techLeadsFixture, got)
	})

	t.Run("missing file errors", func(t *testing.T) {
		_, err := loadTechLeads(filepath.Join("testdata", "techleads", "does-not-exist"))
		require.Error(t, err)
	})

	t.Run("missing alias errors loudly", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "OWNERS_ALIASES")
		require.NoError(t, os.WriteFile(path, []byte("aliases:\n  some-other-group:\n    - alice\n"), 0o644))
		_, err := loadTechLeads(path)
		require.Error(t, err)
	})

	t.Run("empty alias membership errors loudly", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "OWNERS_ALIASES")
		require.NoError(t, os.WriteFile(path, []byte("aliases:\n  sig-node-tech-leads: []\n"), 0o644))
		_, err := loadTechLeads(path)
		require.Error(t, err)
	})

	t.Run("malformed yaml errors", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "OWNERS_ALIASES")
		// A tab used for indentation is invalid YAML.
		require.NoError(t, os.WriteFile(path, []byte("aliases:\n\tsig-node-tech-leads:\n"), 0o644))
		_, err := loadTechLeads(path)
		require.Error(t, err)
	})
}
