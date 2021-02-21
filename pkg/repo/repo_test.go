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

package repo_test

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/repo"
)

func TestProposalValidate(t *testing.T) {
	testcases := []struct {
		name string
		file string
		err  error
	}{
		{
			name: "valid KEP passes validate",
			file: "testdata/valid-kep.yaml",
			err:  nil,
		},
		{
			name: "invalid KEP fails validate for owning-sig",
			file: "testdata/invalid-kep.yaml",
			err:  fmt.Errorf(`kep is invalid: error validating KEP metadata: "owning-sig" must be one of (committee-code-of-conduct,committee-product-security,committee-steering,sig-api-machinery,sig-apps,sig-architecture,sig-auth,sig-autoscaling,sig-cli,sig-cloud-provider,sig-cluster-lifecycle,sig-contributor-experience,sig-docs,sig-instrumentation,sig-multicluster,sig-network,sig-node,sig-release,sig-scalability,sig-scheduling,sig-security,sig-service-catalog,sig-storage,sig-testing,sig-ui,sig-usability,sig-windows,ug-big-data,ug-vmware-users,wg-api-expression,wg-component-standard,wg-data-protection,wg-iot-edge,wg-k8s-infra,wg-lts,wg-multitenancy,wg-naming,wg-policy,wg-security-audit) but it is a string: sig-awesome`),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := ioutil.ReadFile(tc.file)
			require.NoError(t, err)

			var p api.Proposal
			err = yaml.Unmarshal(b, &p)
			require.NoError(t, err)

			err = p.Validate()
			if tc.err == nil {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.err.Error())
			}
		})
	}
}

func TestFindLocalKEPs(t *testing.T) {
	testcases := []struct {
		sig  string
		keps []string
	}{
		{
			"sig-architecture",
			[]string{"123-newstyle", "20200115-kubectl-diff.md"},
		},
		{
			"sig-sig",
			[]string{},
		},
	}

	r, repoErr := repo.New("testdata")
	require.Nil(t, repoErr)

	for i, tc := range testcases {
		k := r.LoadLocalKEPs(tc.sig)
		if len(k) != len(tc.keps) {
			t.Errorf("Test case %d: expected %d but got %d", i, len(tc.keps), len(k))
			continue
		}

		for j, kn := range k {
			if kn.Name != tc.keps[j] {
				t.Errorf("Test case %d: expected %s but got %s", i, tc.keps[j], kn.Name)
			}
		}
	}
}
