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

type PRRApprovals []*PRRApproval

func (p *PRRApprovals) AddPRRApproval(prrApproval *PRRApproval) {
	*p = append(*p, prrApproval)
}

type PRRApproval struct {
	Number string       `json:"kep-number" yaml:"kep-number"`
	Alpha  PRRMilestone `json:"alpha" yaml:"alpha,omitempty"`
	Beta   PRRMilestone `json:"beta" yaml:"beta,omitempty"`
	Stable PRRMilestone `json:"stable" yaml:"stable,omitempty"`

	Error error `json:"-" yaml:"-"`
}

// TODO(api): Can we refactor the proposal `Milestone` to retrieve this?
type PRRMilestone struct {
	Approver string `json:"approver" yaml:"approver"`
}
