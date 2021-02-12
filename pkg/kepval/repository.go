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
	"github.com/sirupsen/logrus"

	"k8s.io/enhancements/api"
)

const (
	// TODO: Make this configurable and set in a client instead
	DefaultPRRDir  = "prod-readiness"
	kepMetadata    = "kep.yaml"
	kepsReadmePath = "enhancements/keps/README.md"
)

var files = []string{}

// This is the actual validation check of all KEPs in this repo
func ValidateRepository(kepDir string) ([]string, map[string][]error, error) {
	warnings := []string{}
	valErrMap := make(map[string][]error)

	// Find all the KEPs
	err := filepath.Walk(kepDir, walkFn)

	// This indicates a problem walking the filepath, not a validation error.
	if err != nil {
		return warnings, valErrMap, errors.Wrap(err, "walking repository")
	}

	if len(files) == 0 {
		return warnings, valErrMap, errors.New("must find more than zero KEPs")
	}

	kepHandler, err := api.NewKEPHandler()
	if err != nil {
		return warnings, valErrMap, errors.Wrap(err, "creating KEP handler")
	}

	prrHandler, err := api.NewPRRHandler()
	if err != nil {
		return warnings, valErrMap, errors.Wrap(err, "creating PRR handler")
	}

	prrDir := filepath.Join(kepDir, DefaultPRRDir)
	logrus.Infof("PRR directory: %s", prrDir)

	for _, filename := range files {
		kepFile, err := os.Open(filename)
		if err != nil {
			return warnings, valErrMap, errors.Wrapf(err, "could not open file %s", filename)
		}

		defer kepFile.Close()

		logrus.Infof("parsing %s", filename)
		kep, kepParseErr := kepHandler.Parse(kepFile)
		if kepParseErr != nil {
			return warnings, valErrMap, errors.Wrap(kepParseErr, "parsing KEP file")
		}

		// TODO: This shouldn't be required once we push the errors into the
		//       parser struct
		if kep.Error != nil {
			return warnings, valErrMap, errors.Wrapf(kep.Error, "%v has an error", filename)
		}

		err = ValidatePRR(kep, prrHandler, prrDir)
		if err != nil {
			valErrMap[filename] = append(valErrMap[filename], err)
		}
	}

	if len(valErrMap) > 0 {
		for filename, errs := range valErrMap {
			logrus.Infof("the following PRR validation errors occurred in %s:", filename)

			for _, e := range errs {
				logrus.Infof("%v", e)
			}
		}
	}

	return warnings, valErrMap, nil
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
		// Assume symlink from old KEP location to new. The new location
		// will be processed separately, so no need to process it here.
		return nil
	}

	metadataFile := filepath.Join(dir, kepMetadata)
	if _, err := os.Stat(metadataFile); err == nil {
		// There is kep metadata file in this directory, only that one should be processed.
		if info.Name() == kepMetadata {
			files = append(files, metadataFile)
		}
		return nil
	}

	if ignore(dir, info.Name()) {
		return nil
	}

	return fmt.Errorf("incorrect KEP format: %s", path)
}

// TODO: Consider replacing with a .kepignore file
// TODO: Is this a duplicate of the package function?
// ignore certain files in the keps/ subdirectory
func ignore(dir, name string) bool {
	if !strings.HasSuffix(name, "md") {
		return true
	}

	return strings.HasSuffix(filepath.Join(dir, name), kepsReadmePath) ||	name == "FAQ.md"
}
