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

package api

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"

	"github.com/pkg/errors"

	"sigs.k8s.io/yaml"
)

// GroupFetcher is responsible for getting informationg about groups in the
// Kubernetes Community
type GroupFetcher interface {
	// FetchGroups returns the list of valid Kubernetes Community groups
	// e.g. (SIGs, WGs, UGs, Committees)
	FetchGroups() ([]string, error)
	// FetchPRRApprovers returns the list of valid PRR Approvers
	FetchPRRApprovers() ([]string, error)
}

// DefaultGroupFetcher returns the default GroupFetcher, which depends on GitHub
func DefaultGroupFetcher() GroupFetcher {
	return &RemoteGroupFetcher{
		GroupsListURL:     "https://raw.githubusercontent.com/kubernetes/community/master/sigs.yaml",
		OwnersAliasesURL:  "https://raw.githubusercontent.com/kubernetes/enhancements/master/OWNERS_ALIASES",
		PRRApproversAlias: "prod-readiness-approvers",
	}
}

// MockGroupFetcher returns the given Groups and PRR Approvers
type MockGroupFetcher struct {
	Groups       []string
	PRRApprovers []string
}

var _ GroupFetcher = &MockGroupFetcher{}

func NewMockGroupFetcher(groups, prrApprovers []string) GroupFetcher {
	return &MockGroupFetcher{Groups: groups, PRRApprovers: prrApprovers}
}

func (f *MockGroupFetcher) FetchGroups() ([]string, error) {
	result := make([]string, len(f.Groups))
	copy(result, f.Groups)
	return result, nil
}

func (f *MockGroupFetcher) FetchPRRApprovers() ([]string, error) {
	result := make([]string, len(f.PRRApprovers))
	copy(result, f.PRRApprovers)
	return result, nil
}

// RemoteGroupFetcher returns Groups and PRR Approvers from remote sources
type RemoteGroupFetcher struct {
	// Basically kubernetes/community/sigs.yaml
	GroupsListURL string
	// Basically kubernetes/enhancements/OWNERS_ALIASES
	OwnersAliasesURL string
	// The alias name to look for in OWNERS_ALIASES
	PRRApproversAlias string
}

var _ GroupFetcher = &RemoteGroupFetcher{}

// FetchGroups returns the list of valid Kubernetes Community groups as
// fetched from a remote source
func (f *RemoteGroupFetcher) FetchGroups() ([]string, error) {
	resp, err := http.Get(f.GroupsListURL)
	if err != nil {
		return nil, errors.Wrap(err, "fetching SIG list")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(
			fmt.Sprintf(
				"invalid status code when fetching SIG list: %d",
				resp.StatusCode,
			),
		)
	}

	re := regexp.MustCompile(`- dir: (.*)$`)

	var result []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		match := re.FindStringSubmatch(scanner.Text())
		if len(match) > 0 {
			result = append(result, match[1])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "scanning SIG list")
	}

	sort.Strings(result)

	return result, nil
}

// FetchPRRApprovers returns a list of PRR approvers.
func (f *RemoteGroupFetcher) FetchPRRApprovers() ([]string, error) {
	resp, err := http.Get(f.OwnersAliasesURL)
	if err != nil {
		return nil, errors.Wrap(err, "fetching owners aliases")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(
			fmt.Sprintf(
				"invalid status code when fetching owners aliases: %d",
				resp.StatusCode,
			),
		)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading owners aliases")
	}

	config := &struct {
		Data map[string][]string `json:"aliases,omitempty"`
	}{}
	if err := yaml.Unmarshal(body, config); err != nil {
		return nil, errors.Wrap(err, "unmarshalling owners aliases")
	}

	var result []string
	result = append(result, config.Data[f.PRRApproversAlias]...)

	if len(result) == 0 {
		return nil, errors.New("retrieved zero PRR approvers, which is unexpected")
	}

	sort.Strings(result)

	return result, nil
}
