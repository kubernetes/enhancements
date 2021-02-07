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

package util

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sort"

	"sigs.k8s.io/yaml"
)

var (
	groups       []string
	prrApprovers []string
)

// Groups returns a list of Kubernetes groups (SIGs, Working Groups, User Groups).
func Groups() []string {
	return groups
}

// PRRApprovers returns a list of PRR approvers.
func PRRApprovers() []string {
	return prrApprovers
}

func init() {
	var err error
	groups, err = fetchGroups()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	prrApprovers, err = fetchPRRApprovers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func fetchGroups() ([]string, error) {
	resp, err := http.Get("https://raw.githubusercontent.com/kubernetes/community/master/sigs.yaml")
	if err != nil {
		return nil, fmt.Errorf("unable to fetch list of sigs: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code when fetching list of sigs: %d", resp.StatusCode)
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
		return nil, fmt.Errorf("unable to scan list of sigs: %v", err)
	}
	sort.Strings(result)
	return result, nil
}

func fetchPRRApprovers() ([]string, error) {
	resp, err := http.Get("https://raw.githubusercontent.com/kubernetes/enhancements/master/OWNERS_ALIASES")
	if err != nil {
		return nil, fmt.Errorf("unable to fetch list of aliases: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code when fetching list of aliases: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read aliases content: %v", err)
	}

	config := &struct {
		Data map[string][]string `json:"aliases,omitempty"`
	}{}
	if err := yaml.Unmarshal(body, config); err != nil {
		return nil, fmt.Errorf("unable to read parse aliases content: %v", err)
	}
	var result []string
	result = append(result, config.Data["prod-readiness-approvers"]...)

	sort.Strings(result)
	return result, nil
}
