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

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"k8s.io/enhancements/api"
)

func ValidatePRR(kep *api.Proposal, h *api.PRRHandler, prrDir string) error {
	requiredPRRApproval, _, _, err := isPRRRequired(kep)
	if err != nil {
		return errors.Wrap(err, "checking if PRR is required")
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
		return errors.Wrapf(err, "could not open file %s", prrFilepath)
	}

	// TODO: Create a context to hold the parsers
	prr, prrParseErr := h.Parse(prrFile)
	if prrParseErr != nil {
		return errors.Wrap(prrParseErr, "parsing PRR approval file")
	}

	// TODO: This shouldn't be required once we push the errors into the
	//       parser struct
	if prr.Error != nil {
		return errors.Wrapf(prr.Error, "%v has an error", prrFilepath)
	}

	stagePRRApprover, err := prr.ApproverForStage(kep.Stage)
	if err != nil {
		return errors.Wrapf(err, "getting PRR approver for %s stage", kep.Stage)
	}

	if stagePRRApprover == "" {
		return errors.New("PRR approver cannot be empty")
	}

	stagePRRApprover = strings.TrimPrefix(stagePRRApprover, "@")

	validApprover := api.IsOneOf(stagePRRApprover, h.PRRApprovers)
	if !validApprover {
		return errors.New(
			fmt.Sprintf(
				"this contributor (%s) is not a PRR approver (%v)",
				stagePRRApprover,
				h.PRRApprovers,
			),
		)
	}

	return nil
}

func isPRRRequired(kep *api.Proposal) (required, missingMilestone, missingStage bool, err error) {
	logrus.Debug("checking if PRR is required")

	required = kep.Status == api.ImplementableStatus || kep.Status == api.ImplementedStatus
	missingMilestone = kep.IsMissingMilestone()
	missingStage = kep.IsMissingStage()

	if missingMilestone {
		logrus.Warnf("Missing the latest milestone field: %s", kep.Filename)
		return required, missingMilestone, missingStage, nil
	}

	if missingStage {
		logrus.Warnf("Missing the stage field: %s", kep.Filename)
		return required, missingMilestone, missingStage, nil
	}

	// TODO: Consider making this a function
	prrRequiredAtSemVer, err := semver.ParseTolerant("v1.21")
	if err != nil {
		return required, missingMilestone, missingStage, errors.Wrap(err, "creating a SemVer object for PRRs")
	}

	latestSemVer, err := semver.ParseTolerant(kep.LatestMilestone)
	if err != nil {
		return required, missingMilestone, missingStage, errors.Wrap(err, "creating a SemVer object for latest milestone")
	}

	if latestSemVer.LT(prrRequiredAtSemVer) {
		required = false
		return required, missingMilestone, missingStage, nil
	}

	return required, missingMilestone, missingStage, nil
}
