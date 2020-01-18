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
	"fmt"
	"strings"
)

type KeyMustBeSpecified struct {
	key interface{}
}

func (k *KeyMustBeSpecified) Error() string {
	return fmt.Sprintf("missing key %[1]v", k.key)
}

type KeyMustBeString struct {
	key interface{}
}

func (k *KeyMustBeString) Error() string {
	return fmt.Sprintf("key %[1]v must be a string but it is a %[1]T", k.key)
}

type ValueMustBeString struct {
	key   string
	value interface{}
}

func (v *ValueMustBeString) Error() string {
	return fmt.Sprintf("%q must be a string but it is a %T: %v", v.key, v.value, v.value)
}

type ValueMustBeListOfStrings struct {
	key   string
	value interface{}
}

func (v *ValueMustBeListOfStrings) Error() string {
	return fmt.Sprintf("%q must be a list of strings: %v", v.key, v.value)
}

type MustHaveOneValue struct {
	key string
}

func (m *MustHaveOneValue) Error() string {
	return fmt.Sprintf("%q must have a value", m.key)
}

type MustHaveAtLeastOneValue struct {
	key string
}

func (m *MustHaveAtLeastOneValue) Error() string {
	return fmt.Sprintf("%q must have at least one value", m.key)
}

var mandatoryKeys = []string{"title", "owning-sig"}

func ValidateStructure(parsed map[interface{}]interface{}) error {
	for _, key := range mandatoryKeys {
		if _, found := parsed[key]; !found {
			return &KeyMustBeSpecified{key}
		}
	}

	for key, value := range parsed {
		// First off the key has to be a string. fact.
		k, ok := key.(string)
		if !ok {
			return &KeyMustBeString{k}
		}
		empty := value == nil

		// figure out the types
		switch strings.ToLower(k) {
		// optional strings
		case "editor":
			if empty {
				continue
			}
			fallthrough
		case "title", "owning-sig", "status", "creation-date", "last-updated":
			switch v := value.(type) {
			case []interface{}:
				return &ValueMustBeString{k, v}
			}
			v, ok := value.(string)
			if ok && v == "" {
				return &MustHaveOneValue{k}
			}
			if !ok {
				return &ValueMustBeString{k, v}
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
			switch v := value.(type) {
			case []interface{}:
				if len(v) == 0 {
					return &MustHaveAtLeastOneValue{k}
				}
			case interface{}:
				return &ValueMustBeListOfStrings{k, v}
			}
		}
	}
	return nil
}
