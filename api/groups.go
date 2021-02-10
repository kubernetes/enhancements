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

const (
	// URLs
	sigListURL     = "https://raw.githubusercontent.com/kubernetes/community/master/sigs.yaml"
	ownersAliasURL = "https://raw.githubusercontent.com/kubernetes/enhancements/master/OWNERS_ALIASES"

	// Aliases
	prrApproversAlias = "prod-readiness-approvers"
)

// FetchGroups returns a list of Kubernetes governance groups
// (SIGs, Working Groups, User Groups).
func FetchGroups() ([]string, error) {
	resp, err := http.Get(sigListURL)
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
func FetchPRRApprovers() ([]string, error) {
	resp, err := http.Get(ownersAliasURL)
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
	result = append(result, config.Data[prrApproversAlias]...)

	if len(result) == 0 {
		return nil, errors.New("retrieved zero PRR approvers, which is unexpected")
	}

	sort.Strings(result)

	return result, nil
}
