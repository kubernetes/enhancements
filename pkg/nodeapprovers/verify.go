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

// Package nodeapprovers verifies that reviewers/approvers annotated in a
// kep.yaml with the inline comments "# sig-node-assigned-reviewer" and
// "# sig-node-assigned-approver" are listed in the neighboring OWNERS file
// (assigned reviewers under reviewers:, assigned approvers under approvers:).
package nodeapprovers

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Hardcoded inline-comment markers identifying SIG Node assigned reviewers and
// approvers in a kep.yaml file.
const (
	reviewerMarker = "sig-node-assigned-reviewer"
	approverMarker = "sig-node-assigned-approver"

	reviewerRole = "reviewer"
	approverRole = "approver"
)

// Violation describes a single assigned reviewer/approver that is not correctly
// listed in the neighboring OWNERS file.
type Violation struct {
	// KEPPath is the path to the kep.yaml that carries the assignment.
	KEPPath string
	// Role is either "reviewer" or "approver".
	Role string
	// User is the normalized (lowercased, no leading "@") username.
	User string
	// Reason is a human-readable explanation of the violation.
	Reason string
}

// String returns a readable, single-line representation of the violation.
func (v Violation) String() string {
	return fmt.Sprintf("%s: assigned %s %q %s", v.KEPPath, v.Role, v.User, v.Reason)
}

// ownersFile is the minimal shape of an OWNERS file needed for verification.
type ownersFile struct {
	Reviewers []string `yaml:"reviewers"`
	Approvers []string `yaml:"approvers"`
}

// normalizeUser trims a leading "@" and lowercases the handle. GitHub handles
// are case-insensitive, and OWNERS files use bare names while kep.yaml uses
// "@name".
func normalizeUser(name string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "@"))
}

// markerOf returns the trimmed marker text from a yaml LineComment (stripping a
// leading "#" and surrounding spaces), e.g. "# sig-node-assigned-reviewer" ->
// "sig-node-assigned-reviewer".
func markerOf(lineComment string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lineComment), "#"))
}

// assignedUsers walks the given sequence node (the value of reviewers: or
// approvers:) and returns the normalized usernames whose inline comment matches
// the supplied marker.
func assignedUsers(seq *yaml.Node, marker string) []string {
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return nil
	}

	var users []string
	for _, item := range seq.Content {
		if item.Kind != yaml.ScalarNode {
			continue
		}
		if markerOf(item.LineComment) == marker {
			users = append(users, normalizeUser(item.Value))
		}
	}

	return users
}

// mappingValue returns the value node for the given key in a mapping node, or
// nil if the key is absent or the node is not a mapping.
func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}

	return nil
}

// containsNormalized reports whether the normalized list includes the normalized user.
func containsNormalized(list []string, user string) bool {
	for _, item := range list {
		if normalizeUser(item) == user {
			return true
		}
	}

	return false
}

// VerifyKEP verifies a single kep.yaml file and returns any violations found.
func VerifyKEP(kepYAMLPath string) ([]Violation, error) {
	data, err := os.ReadFile(kepYAMLPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", kepYAMLPath, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", kepYAMLPath, err)
	}

	// Unwrap the document node to get the top-level mapping.
	var root *yaml.Node
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		root = doc.Content[0]
	} else {
		root = &doc
	}

	assignedReviewers := assignedUsers(mappingValue(root, "reviewers"), reviewerMarker)
	assignedApprovers := assignedUsers(mappingValue(root, "approvers"), approverMarker)

	ownersPath := filepath.Join(filepath.Dir(kepYAMLPath), "OWNERS")
	ownersData, err := os.ReadFile(ownersPath)
	if os.IsNotExist(err) {
		if len(assignedReviewers) == 0 && len(assignedApprovers) == 0 {
			// No annotations and no OWNERS file — nothing to check.
			return nil, nil
		}
		var violations []Violation
		for _, u := range assignedReviewers {
			violations = append(violations, Violation{
				KEPPath: kepYAMLPath,
				Role:    reviewerRole,
				User:    u,
				Reason:  "OWNERS file not found",
			})
		}
		for _, u := range assignedApprovers {
			violations = append(violations, Violation{
				KEPPath: kepYAMLPath,
				Role:    approverRole,
				User:    u,
				Reason:  "OWNERS file not found",
			})
		}
		return violations, nil
	} else if err != nil {
		return nil, fmt.Errorf("reading %s: %w", ownersPath, err)
	}

	var owners ownersFile
	if err := yaml.Unmarshal(ownersData, &owners); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", ownersPath, err)
	}

	var violations []Violation
	for _, u := range assignedReviewers {
		if !containsNormalized(owners.Reviewers, u) {
			violations = append(violations, Violation{
				KEPPath: kepYAMLPath,
				Role:    reviewerRole,
				User:    u,
				Reason:  "not listed under reviewers in OWNERS",
			})
		}
	}
	for _, u := range assignedApprovers {
		if !containsNormalized(owners.Approvers, u) {
			violations = append(violations, Violation{
				KEPPath: kepYAMLPath,
				Role:    approverRole,
				User:    u,
				Reason:  "not listed under approvers in OWNERS",
			})
		}
	}

	// Reverse check: every entry in OWNERS must have a corresponding
	// annotation in kep.yaml.
	for _, u := range owners.Reviewers {
		if !containsNormalized(assignedReviewers, normalizeUser(u)) {
			violations = append(violations, Violation{
				KEPPath: kepYAMLPath,
				Role:    reviewerRole,
				User:    normalizeUser(u),
				Reason:  "listed in OWNERS but not annotated as sig-node-assigned-reviewer in kep.yaml",
			})
		}
	}
	for _, u := range owners.Approvers {
		if !containsNormalized(assignedApprovers, normalizeUser(u)) {
			violations = append(violations, Violation{
				KEPPath: kepYAMLPath,
				Role:    approverRole,
				User:    normalizeUser(u),
				Reason:  "listed in OWNERS but not annotated as sig-node-assigned-approver in kep.yaml",
			})
		}
	}

	return violations, nil
}

// VerifyAll walks rootDir (typically the repo's keps/ directory) for files named
// kep.yaml and aggregates the violations from each.
func VerifyAll(rootDir string) ([]Violation, error) {
	var violations []Violation

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "kep.yaml" {
			return nil
		}

		v, err := VerifyKEP(path)
		if err != nil {
			return err
		}
		violations = append(violations, v...)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return violations, nil
}
