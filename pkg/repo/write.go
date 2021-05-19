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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"k8s.io/enhancements/api"
)

func (r *Repo) WriteKEP(kep *api.Proposal) error {
	b, err := yaml.Marshal(kep)
	if err != nil {
		return fmt.Errorf("KEP is invalid: %s", err)
	}

	sig := kep.OwningSIG
	kepName := kep.Name

	if sig == "" {
		return errors.New("owning SIG must be populated")
	}

	if kepName == "" {
		return errors.New("KEP name must be populated")
	}

	kepPath := filepath.Join(r.ProposalPath, sig, kepName)
	logrus.Infof("creating KEP directory: %s", kepPath)
	if err = os.MkdirAll(kepPath, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create KEP path %s: %w", kepPath, err)
	}

	kepYamlPath := filepath.Join(kepPath, ProposalMetadataFilename)

	logrus.Infof("writing KEP metadata to %s", kepYamlPath)

	return ioutil.WriteFile(kepYamlPath, b, os.ModePerm)
}
