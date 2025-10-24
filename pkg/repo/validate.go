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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"k8s.io/enhancements/pkg/kepval"
)

// This is the actual validation check of all KEPs in this repo
func (r *Repo) Validate() (
	warnings []string,
	valErrMap map[string][]error,
	err error,
) {
	valErrMap = map[string][]error{}

	if r.ProposalPath == "" {
		return warnings, valErrMap, errors.New("proposal path cannot be empty")
	}

	kepDir := r.ProposalPath
	files := []string{}

	// Find all the KEPs
	err = filepath.Walk(
		kepDir,
		func(path string, info os.FileInfo, err error) error {
			logrus.Debugf("processing filename %s", info.Name())

			if err != nil {
				return err
			}

			if info.IsDir() {
				if info.Name() == PRRApprovalPathStub {
					return filepath.SkipDir
				}

				return nil
			}

			dir := filepath.Dir(path)

			metadataFilename := ProposalMetadataFilename
			metadataFilepath := filepath.Join(dir, metadataFilename)

			if _, err := os.Stat(metadataFilepath); err == nil {
				// There is KEP metadata file in this directory, only that one should be processed.
				if info.Name() == metadataFilename {
					files = append(files, metadataFilepath)
					return filepath.SkipDir
				}
			}

			if info.Name() == ProposalFilename && dir != kepDir {
				// There is a proposal, we require metadata file for it.
				if _, err := os.Stat(metadataFilepath); os.IsNotExist(err) {
					kepErr := fmt.Errorf("metadata file missing for KEP: %s", metadataFilepath)
					valErrMap[metadataFilepath] = append(valErrMap[metadataFilepath], kepErr)
				}
			}

			return nil
		},
	)

	// This indicates a problem walking the filepath, not a validation error.
	if err != nil {
		return warnings, valErrMap, fmt.Errorf("walking repository: %w", err)
	}

	if len(files) == 0 {
		return warnings, valErrMap, errors.New("must find more than zero KEPs")
	}

	prrDir := r.PRRApprovalPath
	logrus.Infof("PRR directory: %s", prrDir)

	for _, filename := range files {
		if err := validateFile(r, prrDir, filename); err != nil {
			fvErr := &fatalValidationError{}
			if errors.As(err, fvErr) {
				return warnings, valErrMap, err
			}
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

// fatalValidationError will short-circuit KEP parsing and return early.
type fatalValidationError struct{ Err error }

func (e fatalValidationError) Error() string { return e.Err.Error() }
func (e fatalValidationError) Unwrap() error { return e.Err }

// validateFile runs a validation and returns an error if validation fails.
// fatalValidationError will be returned if further parsing should be stopped.
func validateFile(r *Repo, prrDir, filename string) error {
	kepFile, err := os.Open(filename)
	if err != nil {
		return &fatalValidationError{Err: fmt.Errorf("could not open file %s: %w", filename, err)}
	}
	defer kepFile.Close()

	logrus.Infof("parsing %s", filename)
	kepHandler, prrHandler := r.KEPHandler, r.PRRHandler
	kep, kepParseErr := kepHandler.Parse(kepFile)
	if kepParseErr != nil {
		return fmt.Errorf("parsing KEP file: %w", kepParseErr)
	}
	kep.Filename = filename

	// TODO: This shouldn't be required once we push the errors into the
	//       parser struct
	if kep.Error != nil {
		return &fatalValidationError{Err: fmt.Errorf("%v has an error: %w", filename, kep.Error)}
	}

	return kepval.ValidatePRR(kep, prrHandler, prrDir)
}
