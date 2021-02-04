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

package kepval_test

/*
var err = errors.New("")

func TestValidatePRR(t *testing.T) {
	testcases := []struct {
		name        string
		packageName string
		version     string
		kubeVersion string
		expected    string
	}{
		{
			name:        "Kubernetes version supplied",
			kubeVersion: "1.17.0",
			expected:    "1.17.0",
		},
		{
			name:        "Kubernetes version prefixed",
			kubeVersion: "v1.17.0",
			expected:    "1.17.0",
		},
		{
			name:     "Kubernetes version not supplied",
			expected: "",
		},
		{
			name:        "CNI version",
			packageName: "kubernetes-cni",
			version:     "0.8.6",
			kubeVersion: "1.17.0",
			expected:    "0.8.6",
		},
		{
			name:        "CRI tools version",
			packageName: "cri-tools",
			kubeVersion: "1.17.0",
			expected:    "1.17.0",
		},
	}

	//sut, _ := newSUT(nil)
	for _, tc := range testcases {
		actual, err := kepval.ValidatePRR()

		require.Nil(t, err)
		require.Equal(t, tc.expected, actual)
	}
}
*/
