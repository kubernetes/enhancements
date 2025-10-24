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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/sirupsen/logrus"

	"k8s.io/enhancements/api"
)

func ValidatePRR(kep *api.Proposal, h *api.PRRHandler, prrDir string) error {
	requiredPRRApproval, err := isPRRRequired(kep)
	if err != nil {
		return fmt.Errorf("checking if PRR is required: %w", err)
	}

	if !requiredPRRApproval {
		logrus.Debugf("PRR review is not required for %s", kep.Number)
		return nil
	}

	prrFilename := kep.Number + ".yaml"
	prrFilepath := filepath.Join(
		prrDir,
		kep.OwningSIG,
		prrFilename,
	)

	logrus.Infof("PRR file: %s", prrFilepath)

	prrFile, err := os.Open(prrFilepath)
	if os.IsNotExist(err) {
		return err
	}

	if err != nil {
		return fmt.Errorf("could not open file %s: %w", prrFilepath, err)
	}

	// TODO: Create a context to hold the parsers
	prr, prrParseErr := h.Parse(prrFile)
	if prrParseErr != nil {
		return fmt.Errorf("parsing PRR approval file: %w", prrParseErr)
	}

	// TODO: This shouldn't be required once we push the errors into the
	//       parser struct
	if prr.Error != nil {
		return fmt.Errorf("%v has an error: %w", prrFilepath, prr.Error)
	}

	stagePRRApprover, err := prr.ApproverForStage(kep.Stage)
	if err != nil {
		return fmt.Errorf("getting PRR approver for %s stage: %w", kep.Stage, err)
	}

	if stagePRRApprover == "" {
		return errors.New("PRR approver cannot be empty")
	}

	if strings.HasPrefix(stagePRRApprover, "@") {
		stagePRRApprover = strings.TrimPrefix(stagePRRApprover, "@")
	}

	validApprover := api.IsOneOf(stagePRRApprover, h.PRRApprovers)
	if !validApprover {
		return fmt.Errorf("this contributor (%s) is not a PRR approver (%v)", stagePRRApprover, h.PRRApprovers)
	}

	return nil
}

func isPRRRequired(kep *api.Proposal) (required bool, err error) {
	logrus.Debug("checking if PRR is required")

	required = kep.Status == api.ImplementableStatus || kep.Status == api.ImplementedStatus
	if !required {
		return false, nil
	}

	if kep.IsMissingMilestone() {
		return required, fmt.Errorf("missing the latest milestone field: %s", kep.Filename)
	}

	if kep.IsMissingStage() {
		return required, fmt.Errorf("missing the stage field: %s", kep.Filename)
	}

	// TODO: Consider making this a function
	prrRequiredAtSemVer, err := semver.ParseTolerant("v1.21")
	if err != nil {
		return required, fmt.Errorf("creating a SemVer object for PRRs: %w", err)
	}

	latestSemVer, err := semver.ParseTolerant(kep.LatestMilestone)
	if err != nil {
		return required, fmt.Errorf("creating a SemVer object for latest milestone: %w", err)
	}

	if latestSemVer.LT(prrRequiredAtSemVer) {
		required = false
		return required, nil
	}

	return required, nil
}
