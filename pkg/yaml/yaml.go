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

package yaml

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

// UnmarshalStrict unmarshals the contents of body into the given interface,
// and returns an error if any duplicate or unknown fields are encountered
func UnmarshalStrict(body []byte, v interface{}) error {
	r := bytes.NewReader(body)
	d := yaml.NewDecoder(r)
	d.KnownFields(true)
	err := d.Decode(v)
	return err
}

// Marshal returns a byte array containing a YAML representation of the
// given interface, and a non-nil error if there was an error
func Marshal(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}
