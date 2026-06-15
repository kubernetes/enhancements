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
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// minEnforcedMinor is the minimum Kubernetes minor version (in latest-milestone)
// for which tech-lead approver rules are enforced. KEPs whose latest-milestone
// is before v1.<minEnforcedMinor> are grandfathered.
const minEnforcedMinor = 36

// Hardcoded inline-comment markers identifying SIG Node assigned reviewers and
// approvers in a kep.yaml file.
const (
	reviewerMarker = "sig-node-assigned-reviewer"
	approverMarker = "sig-node-assigned-approver"

	reviewerRole = "reviewer"
	approverRole = "approver"

	// techLeadsAlias is the OWNERS_ALIASES group name whose members are SIG
	// Node tech leads.
	techLeadsAlias = "sig-node-tech-leads"
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
	// File-level violations (e.g. "no approvers listed") carry no specific user;
	// omit the empty handle so the message reads cleanly.
	if v.User == "" {
		return fmt.Sprintf("%s: %s %s", v.KEPPath, v.Role, v.Reason)
	}
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

// parseKEPRoot reads a kep.yaml file and returns its top-level mapping node,
// unwrapping the surrounding document node. Callers must tolerate a non-mapping
// node (e.g. for an empty file); mappingValue returns nil for any lookup in that
// case.
func parseKEPRoot(kepYAMLPath string) (*yaml.Node, error) {
	data, err := os.ReadFile(kepYAMLPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", kepYAMLPath, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", kepYAMLPath, err)
	}

	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0], nil
	}

	return &doc, nil
}

// walkKEPs walks rootDir for files named kep.yaml and aggregates the violations
// returned by verify for each.
func walkKEPs(rootDir string, verify func(path string) ([]Violation, error)) ([]Violation, error) {
	var violations []Violation

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "kep.yaml" {
			return nil
		}

		v, err := verify(path)
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
	root, err := parseKEPRoot(kepYAMLPath)
	if err != nil {
		return nil, err
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
	return walkKEPs(rootDir, VerifyKEP)
}

// ownersAliasesFile is the minimal shape of an OWNERS_ALIASES file needed to
// resolve the sig-node-tech-leads group.
type ownersAliasesFile struct {
	Aliases map[string][]string `yaml:"aliases"`
}

// loadTechLeads parses an OWNERS_ALIASES file and returns the set of normalized
// usernames belonging to the sig-node-tech-leads group. It returns an error if
// the alias is absent, so a misconfigured OWNERS_ALIASES fails loudly.
func loadTechLeads(ownersAliasesPath string) (map[string]bool, error) {
	data, err := os.ReadFile(ownersAliasesPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", ownersAliasesPath, err)
	}

	var aliases ownersAliasesFile
	if err := yaml.Unmarshal(data, &aliases); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", ownersAliasesPath, err)
	}

	members, ok := aliases.Aliases[techLeadsAlias]
	if !ok {
		return nil, fmt.Errorf("%s: alias %q not found", ownersAliasesPath, techLeadsAlias)
	}

	techLeads := make(map[string]bool, len(members))
	for _, m := range members {
		techLeads[normalizeUser(m)] = true
	}

	// An empty membership set would silently let any KEP listing the
	// "@sig-node-tech-leads" alias pass while no real tech lead is enforced;
	// fail loudly instead.
	if len(techLeads) == 0 {
		return nil, fmt.Errorf("%s: alias %q has no members", ownersAliasesPath, techLeadsAlias)
	}

	return techLeads, nil
}

// approverEntry is a single approver listed under approvers: in a kep.yaml,
// with its normalized handle and whether it carries the
// "# sig-node-assigned-approver" inline marker.
type approverEntry struct {
	User   string
	Marked bool
}

// parseMilestoneMinor extracts the minor version number from a milestone string
// like "v1.36" or "v1.22". Returns 0 if the string is empty or unparseable.
func parseMilestoneMinor(milestone string) int {
	milestone = strings.TrimSpace(strings.Trim(milestone, "\""))
	milestone = strings.TrimPrefix(milestone, "v")
	parts := strings.SplitN(milestone, ".", 3)
	if len(parts) < 2 {
		return 0
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return minor
}

// kepMetadata holds the parsed fields from a kep.yaml needed for verification.
type kepMetadata struct {
	status          string
	stage           string
	latestMilestone string
	entries         []approverEntry
}

// parseKEPMetadata parses a kep.yaml and returns its stage, latest-milestone,
// and approvers sequence.
func parseKEPMetadata(kepYAMLPath string) (kepMetadata, error) {
	root, err := parseKEPRoot(kepYAMLPath)
	if err != nil {
		return kepMetadata{}, err
	}

	var meta kepMetadata
	if statusNode := mappingValue(root, "status"); statusNode != nil && statusNode.Kind == yaml.ScalarNode {
		meta.status = strings.TrimSpace(statusNode.Value)
	}
	if stageNode := mappingValue(root, "stage"); stageNode != nil && stageNode.Kind == yaml.ScalarNode {
		meta.stage = strings.TrimSpace(stageNode.Value)
	}
	if msNode := mappingValue(root, "latest-milestone"); msNode != nil && msNode.Kind == yaml.ScalarNode {
		meta.latestMilestone = strings.TrimSpace(msNode.Value)
	}

	approvers := mappingValue(root, "approvers")
	if approvers == nil || approvers.Kind != yaml.SequenceNode {
		return meta, nil
	}

	for _, item := range approvers.Content {
		if item.Kind != yaml.ScalarNode {
			continue
		}
		meta.entries = append(meta.entries, approverEntry{
			User:   normalizeUser(item.Value),
			Marked: markerOf(item.LineComment) == approverMarker,
		})
	}

	return meta, nil
}

// VerifyTechLeadApprovers verifies that a single kep.yaml lists an acceptable
// approver per the stage-dependent rules. upcomingMinor is the minor version of
// the upcoming Kubernetes release (e.g. 37 for v1.37) and determines whether an
// alpha KEP is actively in alpha or from a past cycle.
func VerifyTechLeadApprovers(kepYAMLPath string, techLeads map[string]bool, upcomingMinor int) ([]Violation, error) {
	meta, err := parseKEPMetadata(kepYAMLPath)
	if err != nil {
		return nil, err
	}

	// Skip KEPs that are no longer active (withdrawn, rejected, replaced).
	switch meta.status {
	case "withdrawn", "rejected", "replaced":
		return nil, nil
	}

	// Skip KEPs whose latest-milestone predates the enforcement threshold.
	// A missing or unparseable milestone (minor == 0) is also skipped, as
	// those are legacy KEPs that predate the policy.
	if minor := parseMilestoneMinor(meta.latestMilestone); minor < minEnforcedMinor {
		return nil, nil
	}

	entries := meta.entries
	stage := meta.stage

	if len(entries) == 0 {
		return []Violation{{
			KEPPath: kepYAMLPath,
			Role:    approverRole,
			Reason:  "no approvers listed",
		}}, nil
	}

	hasTechLead := false
	hasMarked := false
	for _, e := range entries {
		if techLeads[e.User] {
			hasTechLead = true
		}
		if e.Marked {
			hasMarked = true
		}
	}

	// A KEP is "actively alpha" only if its latest-milestone is in the
	// upcoming release or later. KEPs from past cycles may already have
	// assigned-approver markers in preparation for the next stage.
	activelyAlpha := stage == "alpha" && parseMilestoneMinor(meta.latestMilestone) >= upcomingMinor

	var violations []Violation
	if activelyAlpha {
		if !hasTechLead {
			violations = append(violations, Violation{
				KEPPath: kepYAMLPath,
				Role:    approverRole,
				Reason:  "alpha-stage KEP must list at least one sig-node-tech-leads member as approver",
			})
		}
		for _, e := range entries {
			if e.Marked {
				violations = append(violations, Violation{
					KEPPath: kepYAMLPath,
					Role:    approverRole,
					User:    e.User,
					Reason:  "alpha-stage KEP must not use # sig-node-assigned-approver marker",
				})
			}
		}
		return violations, nil
	}

	if !hasTechLead && !hasMarked {
		violations = append(violations, Violation{
			KEPPath: kepYAMLPath,
			Role:    approverRole,
			Reason:  "non-alpha KEP must list a sig-node-tech-leads member or an approver marked # sig-node-assigned-approver",
		})
	}

	return violations, nil
}

// VerifyAllTechLeadApprovers loads the sig-node-tech-leads set from
// ownersAliasesPath, then walks kepsRootDir for kep.yaml files and aggregates
// the tech-lead approver violations from each. upcomingMinor is the minor
// version of the upcoming Kubernetes release (see FetchUpcomingMinor).
func VerifyAllTechLeadApprovers(kepsRootDir, ownersAliasesPath string, upcomingMinor int) ([]Violation, error) {
	techLeads, err := loadTechLeads(ownersAliasesPath)
	if err != nil {
		return nil, err
	}

	return walkKEPs(kepsRootDir, func(path string) ([]Violation, error) {
		return VerifyTechLeadApprovers(path, techLeads, upcomingMinor)
	})
}

// sigReleaseContentsURL is the GitHub API endpoint for listing the releases
// directory in the kubernetes/sig-release repository.
const sigReleaseContentsURL = "https://api.github.com/repos/kubernetes/sig-release/contents/releases"

// FetchUpcomingMinor queries the kubernetes/sig-release GitHub repository to
// determine the upcoming Kubernetes minor version. It lists the release-X.Y
// directories and returns the highest minor version found.
func FetchUpcomingMinor() (int, error) {
	resp, err := http.Get(sigReleaseContentsURL)
	if err != nil {
		return 0, fmt.Errorf("fetching sig-release contents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("fetching sig-release contents: HTTP %d", resp.StatusCode)
	}

	var entries []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return 0, fmt.Errorf("decoding sig-release contents: %w", err)
	}

	maxMinor := 0
	for _, e := range entries {
		name := strings.TrimPrefix(e.Name, "release-")
		if name == e.Name {
			continue // not a release-X.Y directory
		}
		if minor := parseMilestoneMinor(name); minor > maxMinor {
			maxMinor = minor
		}
	}

	if maxMinor == 0 {
		return 0, fmt.Errorf("no release-X.Y directories found in sig-release")
	}

	return maxMinor, nil
}
