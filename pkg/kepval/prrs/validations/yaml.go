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

package validations

import (
	"fmt"
	"sort"
	"strings"

	"k8s.io/enhancements/pkg/kepval/util"
)

var mandatoryKeys = []string{"kep-number"}

func ValidateStructure(parsed map[interface{}]interface{}) error {
	for _, key := range mandatoryKeys {
		if _, found := parsed[key]; !found {
			return util.NewKeyMustBeSpecified(key)
		}
	}

	for key, value := range parsed {
		// First off the key has to be a string. fact.
		k, ok := key.(string)
		if !ok {
			return util.NewKeyMustBeString(k)
		}

		// figure out the types
		switch strings.ToLower(k) {
		case "alpha", "beta", "stable":
			switch v := value.(type) {
			case map[interface{}]interface{}:
				if err := validateMilestone(v); err != nil {
					return fmt.Errorf("invalid %s field: %v", k, err)
				}
			case interface{}:
				return util.NewValueMustBeStruct(k, v)
			}
		}
	}
	return nil
}

func validateMilestone(parsed map[interface{}]interface{}) error {
	// prrApprovers must be sorted to use SearchStrings down below...
	prrApprovers := util.PRRApprovers()
	sort.Strings(prrApprovers)

	for key, value := range parsed {
		// First off the key has to be a string. fact.
		k, ok := key.(string)
		if !ok {
			return util.NewKeyMustBeString(k)
		}

		// figure out the types
		switch strings.ToLower(k) {
		case "approver":
			switch v := value.(type) {
			case []interface{}:
				return util.NewValueMustBeString(k, v)
			}

			// TODO(lint): Error return value is not checked (errcheck)
			//nolint:errcheck
			v, _ := value.(string)
			if len(v) > 0 && v[0] == '@' {
				// If "@" is appended at the beginning, remove it.
				v = v[1:]
			}

			index := sort.SearchStrings(prrApprovers, v)
			if index >= len(prrApprovers) || prrApprovers[index] != v {
				return util.NewValueMustBeOneOf(k, v, prrApprovers)
			}
		}
	}
	return nil
}
