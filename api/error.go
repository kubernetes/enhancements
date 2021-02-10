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

package api

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	// PRR errors

	ErrPRRMilestonesAllEmpty error = errors.New(
		"none of the PRR milestones are populated",
	)

	ErrPRRMilestoneIsNil error = errors.New(
		"the selected PRR milestone stage is not populated",
	)

	ErrPRRApproverUnknown error = errors.New(
		"an unknown error occurred while trying to determine a PRR approver",
	)
)

// KEP errors
func ErrKEPStageIsInvalid(stage string) error {
	return errors.New(
		fmt.Sprintf(
			"the specified stage (%s) should be one of the following: %v",
			stage,
			ValidStages,
		),
	)
}
