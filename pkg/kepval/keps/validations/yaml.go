/*
Copyright 2019 The Kubernetes Authors.

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
	"regexp"
	"sort"
	"strings"

	"k8s.io/enhancements/pkg/kepval/util"
)

var (
	mandatoryKeys = []string{"title", "owning-sig"}
	statuses      = []string{"provisional", "implementable", "implemented", "deferred", "rejected", "withdrawn", "replaced"}
	reStatus      = regexp.MustCompile(strings.Join(statuses, "|"))
	stages        = []string{"alpha", "beta", "stable"}
	reStages      = regexp.MustCompile(strings.Join(stages, "|"))
)

func ValidateStructure(parsed map[interface{}]interface{}) error {
	for _, key := range mandatoryKeys {
		if _, found := parsed[key]; !found {
			return util.NewKeyMustBeSpecified(key)
		}
	}

	listGroups := util.Groups()
	prrApprovers := util.PRRApprovers()

	for key, value := range parsed {
		// First off the key has to be a string. fact.
		k, ok := key.(string)
		if !ok {
			return util.NewKeyMustBeString(k)
		}
		empty := value == nil

		// figure out the types
		switch strings.ToLower(k) {
		case "status":
			switch v := value.(type) {
			case []interface{}:
				return util.NewValueMustBeString(k, v)
			}
			v, _ := value.(string)
			if !reStatus.Match([]byte(v)) {
				return util.NewValueMustBeOneOf(k, v, statuses)
			}
		case "stage":
			switch v := value.(type) {
			case []interface{}:
				return util.NewValueMustBeString(k, v)
			}
			v, _ := value.(string)
			if !reStages.Match([]byte(v)) {
				return util.NewValueMustBeOneOf(k, v, stages)
			}
		case "owning-sig":
			switch v := value.(type) {
			case []interface{}:
				return util.NewValueMustBeString(k, v)
			}
			v, _ := value.(string)
			index := sort.SearchStrings(listGroups, v)
			if index >= len(listGroups) || listGroups[index] != v {
				return util.NewValueMustBeOneOf(k, v, listGroups)
			}
		// optional strings
		case "editor":
			if empty {
				continue
			}
			fallthrough
		case "title", "creation-date", "last-updated":
			switch v := value.(type) {
			case []interface{}:
				return util.NewValueMustBeString(k, v)
			}
			v, ok := value.(string)
			if ok && v == "" {
				return util.NewMustHaveOneValue(k)
			}
			if !ok {
				return util.NewValueMustBeString(k, v)
			}
		// These are optional lists, so skip if there is no value
		case "participating-sigs", "replaces", "superseded-by", "see-also":
			if empty {
				continue
			}
			switch v := value.(type) {
			case []interface{}:
				if len(v) == 0 {
					continue
				}
			case interface{}:
				// This indicates an empty item is valid
				continue
			}
			fallthrough
		case "authors", "reviewers", "approvers":
			switch values := value.(type) {
			case []interface{}:
				if len(values) == 0 {
					return util.NewMustHaveAtLeastOneValue(k)
				}
				if strings.ToLower(k) == "participating-sigs" {
					for _, value := range values {
						v := value.(string)
						index := sort.SearchStrings(listGroups, v)
						if index >= len(listGroups) || listGroups[index] != v {
							return util.NewValueMustBeOneOf(k, v, listGroups)
						}
					}
				}
			case interface{}:
				return util.NewValueMustBeListOfStrings(k, values)
			}
		case "prr-approvers":
			switch values := value.(type) {
			case []interface{}:
				// prrApprovers must be sorted to use SearchStrings down below...
				sort.Strings(prrApprovers)
				for _, value := range values {
					v := value.(string)
					if len(v) > 0 && v[0] == '@' {
						// If "@" is appeneded at the beginning, remove it.
						v = v[1:]
					}

					index := sort.SearchStrings(prrApprovers, v)
					if index >= len(prrApprovers) || prrApprovers[index] != v {
						return util.NewValueMustBeOneOf(k, v, prrApprovers)
					}
				}
			case interface{}:
				return util.NewValueMustBeListOfStrings(k, values)
			}
		}
	}
	return nil
}
