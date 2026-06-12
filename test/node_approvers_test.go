/*
Copyright The Kubernetes Authors.

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

package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"k8s.io/enhancements/pkg/nodeapprovers"
)

// TestNodeApprovers verifies that every kep.yaml reviewer/approver annotated
// with the sig-node-assigned-reviewer / sig-node-assigned-approver inline
// comment is listed in the neighboring OWNERS file. It runs over the real
// keps/ tree and is exercised automatically in CI via hack/test-go.sh.
func TestNodeApprovers(t *testing.T) {
	wd, err := os.Getwd()
	require.Nil(t, err)

	rootDir := filepath.Dir(wd)

	violations, err := nodeapprovers.VerifyAll(filepath.Join(rootDir, "keps"))
	require.Nil(t, err)

	if len(violations) > 0 {
		msgs := make([]string, 0, len(violations))
		for _, v := range violations {
			msgs = append(msgs, v.String())
		}
		t.Fatalf(
			"%d SIG Node assigned reviewer/approver violation(s) found:\n%s",
			len(violations),
			strings.Join(msgs, "\n"),
		)
	}
}
