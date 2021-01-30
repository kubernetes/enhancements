/*
Copyright 2021 The Kubernetes Authors.

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

package repo

import (
	"fmt"
	"testing"
	"reflect"
	"github.com/stretchr/testify/require"
	"sort"
)

func TestValidateQueryOpt(t *testing.T) {
	testcases := []struct {
		name      string
		queryOpts QueryOpts
		err       error
	}{
		{
			name: "Valid SIG",
			queryOpts: QueryOpts{
				Name:       "1011-test",
				Groups:     []string{"sig-multicluster"},
				IncludePRs: true,
				Output:     "json",
			},
			err: nil,
		},
		{
			name: "Invalid SIG",
			queryOpts: QueryOpts{
				Name:       "1011-test-xyz",
				Groups:     []string{"sig-xyz"},
				IncludePRs: true,
				Output:     "json",
			},
			err: fmt.Errorf("No SIG matches any of the passed regular expressions"),
		},
		{
			name: "Unsupported Output format",
			queryOpts: QueryOpts{
				Name:       "1011-test-testing",
				Groups:     []string{"sig-testing"},
				IncludePRs: true,
				Output:     "PDF",
			},
			err: fmt.Errorf("unsupported output format: PDF. Valid values: [table json yaml]"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			queryOpts := tc.queryOpts
			err := queryOpts.Validate()

			if tc.err == nil {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err, tc.err.Error())
			}
		})
	}
}

func TestSliceToMap(t *testing.T) {
	testcases := []struct {
		name      	string
		list 					[]string
		returnValue map[string]bool
	}{
		{
			name: "array to map bool values",
			list: []string{"a","b","c","d"},
			returnValue:  map[string]bool{
				"a": true,
				"b": true,
				"c": true,
				"d": true,
			},
		},
		{
			name: "Empty array to map bool values",
			list: []string{ },
			returnValue:  map[string]bool{ },
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var list = tc.list
			actualReturnValue := sliceToMap(list)
			isEqual := reflect.DeepEqual(actualReturnValue, tc.returnValue)
			if !isEqual {
				t.Errorf("Expected %v but got %v", tc.returnValue, actualReturnValue)
			}

		})
	}
}

func TestSliceContains(t *testing.T) {
	testcases := []struct {
		name      	string
		sourceList 			[]string
		target 		  string
		isPresent   bool
	}{
		{
			name: "Target value is present in sourceList list",
			sourceList: []string{"a","b","c","d"},
			target: "a",
			isPresent: true,
		},
		{
			name: "Target value is not present in sourceList list",
			sourceList: []string{"a","b","c","d"},
			target: "A",
			isPresent: false,
		},
		{
			name: "Target value is not present in sourceList list",
			sourceList: []string{},
			target: "",
			isPresent: false,
		},
		{
			name: "Target value is not present in sourceList list",
			sourceList: []string{"a", "z"},
			target: "z ",
			isPresent: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			isPresent := sliceContains(tc.sourceList, tc.target)
			require.Equal(t, tc.isPresent, isPresent)

		})
	}
}

func TestSelectByRegexp(t *testing.T) {
	testcases := []struct {
		name      	string
		sourceList 			[]string
		targetList 		  []string
		matches   []string
		isError bool
		error error
	}{
		{
			name: "targetList is present in sourceList list",
			sourceList: []string{"a","b","c","d"},
			targetList: []string{"a"},
			matches: []string{"a"},
		},
		{
			name: "targetList with multiple values is present in sourceList list",
			sourceList: []string{"a","b","c","d"},
			targetList: []string{"c", "a"},
			matches: []string{"c", "a"},
		},
		{
			name: "some targetList element is present in sourceList list",
			sourceList: []string{"a","b","c","d"},
			targetList: []string{"c", "e"},
			matches: []string{"c"},
		},
		{
			name: "targetList is not present in sourceList list",
			sourceList: []string{"a","b","c","d"},
			targetList: []string{"A"},
			matches: []string(nil),
		},
		{
			name: "targetList is not present in sourceList list",
			sourceList: []string{},
			targetList: []string{""},
			matches: []string(nil),
		},
		{
			name: "targetList is not present in sourceList list",
			sourceList: []string{"a", "z"},
			targetList: []string{"z "},
			matches: []string(nil),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			matches, error := selectByRegexp(tc.sourceList, tc.targetList)
			require.NoError(t, error)
			sort.Strings(tc.matches)
			sort.Strings(matches)
			require.Equal(t, tc.matches, matches)

		})
	}
}
