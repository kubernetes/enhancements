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

package kepval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"k8s.io/enhancements/api"
)

const (
	prrsDir     = "keps/prod-readiness"
	kepMetadata = "kep.yaml"
)

var files = []string{}

// This is the actual validation check of all KEPs in this repo
func ValidateRepository(kepsDir string) error {
	// Find all the keps
	err := filepath.Walk(
		filepath.Join("..", kepsDir),
		walkFn,
	)

	// This indicates a problem walking the filepath, not a validation error.
	if err != nil {
		return errors.Wrap(err, "walking repository")
	}

	if len(files) == 0 {
		return errors.New("must find more than 0 keps")
	}

	kepHandler := &api.KEPHandler{}
	prrHandler := &api.PRRHandler{}
	prrsDir := filepath.Join("..", prrsDir)

	for _, filename := range files {
		kepFile, err := os.Open(filename)
		if err != nil {
			return errors.Wrapf(err, "could not open file %s", filename)
		}

		defer kepFile.Close()

		kep, kepParseErr := kepHandler.Parse(kepFile)
		if kepParseErr != nil {
			return errors.Wrap(kepParseErr, "parsing KEP file")
		}

		// TODO: This shouldn't be required once we push the errors into the
		//       parser struct
		if kep.Error != nil {
			return errors.Wrapf(kep.Error, "%v has an error", filename)
		}

		requiredPRRApproval := len(kep.Number) > 0 && kep.LatestMilestone >= "v1.21" && kep.Status == "implementable"
		if !requiredPRRApproval {
			return errors.New("needs PRR approval")
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
			// TODO: Is this actually the error we want to return here?
			return needsPRRApproval(stageMilestone, kep.Stage, prrFilename)
		}

		if err != nil {
			return errors.Wrapf(err, "could not open file %s", prrFilename)
		}

		prr, prrParseErr := prrHandler.Parse(prrFile)
		if prrParseErr != nil {
			return errors.Wrap(prrParseErr, "parsing PRR approval file")
		}

		// TODO: This shouldn't be required once we push the errors into the
		//       parser struct
		if prr.Error != nil {
			return errors.Wrapf(prr.Error, "%v has an error", prrFilename)
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
				return needsPRRApproval(stageMilestone, kep.Stage, prrFilename)
			}
		}
	}

	return nil
}

// TODO: Refactor and maybe move into a more suitable package
func needsPRRApproval(milestone, stage, filename string) error {
	return errors.New(
		fmt.Sprintf(
			`PRR approval is required to target milestone %s (stage %s).
For more details about PRR approval see: https://git.k8s.io/kubernetes/community/sig-architecture/production-readiness.md
To get PRR approval modify appropriately file %s and have this approved by PRR team`,
			milestone,
			stage,
			filename,
		),
	)
}

var walkFn = func(path string, info os.FileInfo, err error) error {
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
