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
	"fmt"
	"strings"
)

type KeyMustBeSpecified struct {
	key interface{}
}

func (k *KeyMustBeSpecified) Error() string {
	return fmt.Sprintf("missing key %[1]v", k.key)
}

func NewKeyMustBeSpecified(key interface{}) error {
	return &KeyMustBeSpecified{
		key: key,
	}
}

type KeyMustBeString struct {
	key interface{}
}

func (k *KeyMustBeString) Error() string {
	return fmt.Sprintf("key %[1]v must be a string but it is a %[1]T", k.key)
}

func NewKeyMustBeString(key interface{}) error {
	return &KeyMustBeString{
		key: key,
	}
}

type ValueMustBeBool struct {
	key   string
	value interface{}
}

func (v *ValueMustBeBool) Error() string {
	return fmt.Sprintf("%q must be a bool but it is a %T: %v", v.key, v.value, v.value)
}

func NewValueMustBeBool(key string, value interface{}) error {
	return &ValueMustBeBool{
		key:   key,
		value: value,
	}
}

type ValueMustBeString struct {
	key   string
	value interface{}
}

func (v *ValueMustBeString) Error() string {
	return fmt.Sprintf("%q must be a string but it is a %T: %v", v.key, v.value, v.value)
}

func NewValueMustBeString(key string, value interface{}) error {
	return &ValueMustBeString{
		key:   key,
		value: value,
	}
}

type ValueMustBeOneOf struct {
	key    string
	value  string
	values []string
}

func (v *ValueMustBeOneOf) Error() string {
	return fmt.Sprintf("%q must be one of (%s) but it is a %T: %v", v.key, strings.Join(v.values, ","), v.value, v.value)
}

func NewValueMustBeOneOf(key, value string, values []string) error {
	return &ValueMustBeOneOf{
		key:    key,
		value:  value,
		values: values,
	}
}

type ValueMustBeListOfStrings struct {
	key   string
	value interface{}
}

func (v *ValueMustBeListOfStrings) Error() string {
	return fmt.Sprintf("%q must be a list of strings: %v", v.key, v.value)
}

func NewValueMustBeListOfStrings(key string, value interface{}) error {
	return &ValueMustBeListOfStrings{
		key:   key,
		value: value,
	}
}

type MustHaveOneValue struct {
	key string
}

func (m *MustHaveOneValue) Error() string {
	return fmt.Sprintf("%q must have a value", m.key)
}

func NewMustHaveOneValue(key string) error {
	return &MustHaveOneValue{
		key: key,
	}
}

type MustHaveAtLeastOneValue struct {
	key string
}

func (m *MustHaveAtLeastOneValue) Error() string {
	return fmt.Sprintf("%q must have at least one value", m.key)
}

func NewMustHaveAtLeastOneValue(key string) error {
	return &MustHaveAtLeastOneValue{
		key: key,
	}
}
