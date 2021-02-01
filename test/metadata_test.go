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

package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/enhancements/pkg/legacy/keps"
	"k8s.io/enhancements/pkg/legacy/prrs"
)

const (
	kepsDir     = "keps"
	prrsDir     = "keps/prod-readiness"
	kepMetadata = "kep.yaml"
)

// This is the actual validation check of all keps in this repo
func TestValidation(t *testing.T) {
	// Find all the keps
	files := []string{}
	err := filepath.Walk(
		filepath.Join("..", kepsDir),
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			dir := filepath.Dir(path)
			// true if the file is a symlink
			if info.Mode()&os.ModeSymlink != 0 {
				// assume symlink from old KEP location to new
				newLocation, err := os.Readlink(path)
				if err != nil {
					return err
				}
				files = append(files, filepath.Join(dir, filepath.Dir(newLocation), kepMetadata))
				return nil
			}
			if ignore(dir, info.Name()) {
				return nil
			}
			files = append(files, path)
			return nil
		},
	)
	// This indicates a problem walking the filepath, not a validation error.
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatal("must find more than 0 keps")
	}

	kepParser := &keps.Parser{}
	prrParser := &prrs.Parser{}
	prrsDir := filepath.Join("..", "..", prrsDir)

	for _, filename := range files {
		t.Run(filename, func(t *testing.T) {
			kepFile, err := os.Open(filename)
			if err != nil {
				t.Fatalf("could not open file %s: %v\n", filename, err)
			}
			defer kepFile.Close()

			kep := kepParser.Parse(kepFile)
			if kep.Error != nil {
				t.Errorf("%v has an error: %v", filename, kep.Error)
			}

			requiredPRRApproval := len(kep.Number) > 0 && kep.LatestMilestone >= "v1.21" && kep.Status == "implementable"
			if !requiredPRRApproval {
				return
			}

			var stageMilestone string
			switch kep.Stage {
			case "alpha":
				stageMilestone = kep.Milestone.Alpha
			case "beta":
				stageMilestone = kep.Milestone.Beta
			case "stable":
				stageMilestone = kep.Milestone.Stable
			}

			prrFilename := kep.Number + ".yaml"
			prrFilename = filepath.Join(prrsDir, kep.OwningSIG, prrFilename)
			prrFile, err := os.Open(prrFilename)
			if os.IsNotExist(err) {
				t.Errorf("PRR approval is required to target milestone %v (stage %v)", stageMilestone, kep.Stage)
				t.Errorf("For more details about PRR approval see: https://github.com/kubernetes/community/blob/master/sig-architecture/production-readiness.md")
				t.Errorf("To get PRR approval modify appropriately file %s and have this approved by PRR team", prrFilename)
				return
			}
			if err != nil {
				t.Fatalf("could not open file %s: %v\n", prrFilename, err)
			}
			prr := prrParser.Parse(prrFile)
			if prr.Error != nil {
				t.Errorf("PRR approval file %v has an error: %v", prrFilename, prr.Error)
				return
			}

			var stagePRRApprover string
			switch kep.Stage {
			case "alpha":
				stagePRRApprover = prr.Alpha.Approver
			case "beta":
				stagePRRApprover = prr.Beta.Approver
			case "stable":
				stagePRRApprover = prr.Stable.Approver
			}
			if len(stageMilestone) > 0 && stageMilestone >= "v1.21" {
				// PRR approval is needed.
				if stagePRRApprover == "" {
					t.Errorf("PRR approval is required to target milestone %v (stage %v)", stageMilestone, kep.Stage)
					t.Errorf("For more details about PRR approval see: https://github.com/kubernetes/community/blob/master/sig-architecture/production-readiness.md")
					t.Errorf("To get PRR approval modify appropriately file %s and have this approved by PRR team", prrFilename)
				}
			}
		})
	}
}

// TODO: Consider replacing with a .kepignore file
// TODO: Is this a duplicate of the package function?
// ignore certain files in the keps/ subdirectory
func ignore(dir, name string) bool {
	if dir == "../keps/NNNN-kep-template" {
		return true // ignore the template directory because its metadata file does not use a valid sig name
	}

	if name == kepMetadata {
		return false // always check metadata files
	}

	if !strings.HasSuffix(name, "md") {
		return true
	}

	if name == "0023-documentation-for-images.md" ||
		name == "0004-cloud-provider-template.md" ||
		name == "YYYYMMDD-kep-template.md" ||
		name == "README.md" ||
		name == "kep-faq.md" {
		return true
	}

	return false
}
