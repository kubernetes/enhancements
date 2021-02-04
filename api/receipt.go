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
	"io"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

type Receipts []*Receipt

func (p *Receipts) AddReceipt(receipt *Receipt) {
	*p = append(*p, receipt)
}

type Receipt struct {
	Title     string `json:"title" yaml:"title"`
	Number    string `json:"kep-number" yaml:"kep-number"`
	OwningSIG string `json:"owningSig" yaml:"owning-sig"`

	// TrackingStatus is the currently tracked status for the enhancement
	// TODO: Ideally should be an enum with possible values: proposed, at-risk, accepted, exceptions, removed
	TrackingStatus string `json:"trackingStatus" yaml:"tracking-status"`

	// Stage denotes the graduation stage for the concerned release cycle
	// TODO: should be an enum with possible values: alpha, beta and stable
	Stage string `json:"stage" yaml:"stage"`

	Filename string    `json:"-" yaml:"-"`
	Error    error     `json:"-" yaml:"-"`
	Proposal *Proposal `json:"-" yaml:"-"`
}

// Validate a Receipt
// TODO: Add actual validation annotations to Receipt
func (r *Receipt) Validate() error {
	v := validator.New()
	if err := v.Struct(r); err != nil {
		return errors.Wrap(err, "running validation")
	}

	return nil
}

type ReceiptHandler Parser

// Parse a Receipt file
// TODO: Implement this
func (p *ReceiptHandler) Parse(in io.Reader) (*Receipt, error) {
	return nil, nil
}

// GetProposal returns the KEP associated with the Receipt
// TODO: Implement this
func (r *Receipt) GetProposal() (*Proposal, error) {
	return nil, nil
}
