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

package proposal

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
)

type Options struct {
	KEP    string // KEP name sig-xxx/xxx-name
	Name   string
	Number string
	SIG    string
}

func NewOptions(kep, sig, name, number string) (*Options, error) {
	if kep == "" {
		return nil, errors.New("must provide a path for the new KEP like sig-architecture/0000-new-kep")
	}

	re := regexp.MustCompile(`([a-z\\-]+)/((\\d+)-.+)`)
	matches := re.FindStringSubmatch(kep)

	if matches == nil || len(matches) != 4 {
		return nil, errors.New(
			fmt.Sprintf("invalid KEP name: %s", kep),
		)
	}

	return &Options{
		KEP:    kep,
		SIG:    sig,
		Name:   name,
		Number: number,
	}, nil
}
