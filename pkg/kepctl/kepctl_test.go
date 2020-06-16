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

package kepctl

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"k8s.io/enhancements/pkg/kepval/keps"
)

func TestValidate(t *testing.T) {
	testcases := []struct {
		name string
		file string
		err  string
	}{
		{
			name: "valid kep passes valdiate",
			file: "testdata/valid-kep.yaml",
		},
		{
			name: "invalid kep fails valdiate for owning-sig",
			file: "testdata/invalid-kep.yaml",
			err:  "but it is a string: sig-awesome",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := ioutil.ReadFile(tc.file)
			require.NoError(t, err)
			var p keps.Proposal
			err = yaml.Unmarshal(b, &p)
			require.NoError(t, err)
			err = validateKEP(&p)
			if tc.err == "" {
				assert.NoError(t, err)
			} else {
				assert.Contains(t, err.Error(), tc.err)
			}

		})
	}
}
