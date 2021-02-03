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
	"os"
	"path/filepath"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"

	"k8s.io/enhancements/api"
)

func ValidatePRR(kep *api.Proposal, prrDir string) error {
	handler := &api.PRRHandler{}

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

	// TODO: Create a context to hold the parsers
	prr, prrParseErr := handler.Parse(prrFile)
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

	return nil
}

func isPRRRequired(kep *api.Proposal) (required, missingMilestone bool, err error) {
	required = true
	missingMilestone = false

	if kep.LatestMilestone == "" {
		required = false
		missingMilestone = true

		return required, missingMilestone, nil
	} else {
		// TODO: Consider making this a function
		prrRequiredAtSemVer, err := semver.ParseTolerant("v1.21")
		if err != nil {
			return required, missingMilestone, errors.Wrap(err, "creating a SemVer object for PRRs")
		}

		latestSemVer, err := semver.ParseTolerant(kep.LatestMilestone)
		if err != nil {
			return required, missingMilestone, errors.Wrap(err, "creating a SemVer object for latest milestone")
		}

		if latestSemVer.LTE(prrRequiredAtSemVer) || kep.Status != "implementable" {
			required = false
		}
	}

	return required, missingMilestone, nil
}
