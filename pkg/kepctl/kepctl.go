/*
Copyright 2020 The Kubernetes Authors.

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

package kepctl

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v2"
	"k8s.io/enhancements/pkg/kepval/keps"
)

type CommonArgs struct {
	RepoPath string //override the default settings
	KEP      string //KEP name sig-xxx/xxx-name
	Name     string
	Number   string
	SIG      string
}

func (c *CommonArgs) validateAndPopulateKEP(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("must provide a path for the new KEP like sig-architecture/0000-new-kep")
	}
	if len(args) == 1 {
		kep := args[0]
		re := regexp.MustCompile("([a-z\\-]+)/((\\d+)-.+)")
		matches := re.FindStringSubmatch(kep)
		if matches == nil || len(matches) != 4 {
			return fmt.Errorf("invalid KEP name: %s", kep)
		}
		c.KEP = kep
		c.SIG = matches[1]
		c.Number = matches[3]
		c.Name = matches[2]
	} else if len(args) > 1 {
		return fmt.Errorf("only one positional argument may be specified, the KEP name, but multiple were received: %s", args)
	}
	return nil
}

type Client struct {
	RepoPath string
	In       io.Reader
	Out      io.Writer
	Err      io.Writer
}

func (c *Client) findEnhancementsRepo(opts CommonArgs) (string, error) {
	dir := c.RepoPath
	if opts.RepoPath != "" {
		dir = opts.RepoPath
	}

	fi, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("unable to find enhancements repo: %s", err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("invalid enhancements repo path: %s", dir)
	}
	return dir, nil
}

// New returns a new kepctl client configured to use the the normal os.Stdxxx and Filesystem
func New(repo string) (*Client, error) {
	var err error
	if repo == "" {
		repo, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("unable to determine enhancements repo path: %s", err)
		}
	}
	// build a default client with normal os.Stdxx and Filesystem access. Tests can build their own
	// with appropriate test objects
	return &Client{
		RepoPath: repo,
		In:       os.Stdin,
		Out:      os.Stdout,
		Err:      os.Stderr,
	}, nil
}

// getKepTemplate reads the kep.yaml template from the local
// (per c.RepoPath) k/enhancements, but this could be replaced with a
// template via packr or fetched from Github?
func (c *Client) getKepTemplate(repoPath string) (*keps.Proposal, error) {
	var p keps.Proposal
	path := filepath.Join(repoPath, "keps", "NNNN-kep-template", "kep.yaml")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't find KEP template: %s", err)
	}
	err = yaml.Unmarshal(b, &p)
	if err != nil {
		return nil, fmt.Errorf("invalid KEP template: %s", err)
	}
	return &p, nil
}

// getReadmeTemplate reads from the local (per c.RepoPath) k/enhancements, but this
// could be replaced with a template via packr or fetched from Github?
func (c *Client) getReadmeTemplate(repoPath string) ([]byte, error) {
	path := filepath.Join(repoPath, "keps", "NNNN-kep-template", "README.md")
	return ioutil.ReadFile(path)
}

func validateKEP(p *keps.Proposal) error {
	b, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	r := bytes.NewReader(b)
	parser := &keps.Parser{}

	kep := parser.Parse(r)
	if kep.Error != nil {
		return fmt.Errorf("kep is invalid: %s", kep.Error)
	}
	return nil
}

func (c *Client) readKEP(repoPath string, sig, name string) (*keps.Proposal, error) {
	kepPath := filepath.Join(
		repoPath,
		"keps",
		sig,
		name,
		"kep.yaml")
	b, err := ioutil.ReadFile(kepPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read KEP metadata: %s", err)
	}
	var p keps.Proposal
	err = yaml.Unmarshal(b, &p)
	if err != nil {
		return nil, fmt.Errorf("unable to load KEP metadata: %s", err)
	}
	return &p, nil
}

func (c *Client) writeKEP(kep *keps.Proposal, opts CommonArgs) error {
	path, err := c.findEnhancementsRepo(opts)
	if err != nil {
		return fmt.Errorf("unable to write KEP: %s", err)
	}
	b, err := yaml.Marshal(kep)
	if err != nil {
		return fmt.Errorf("KEP is invalid: %s", err)
	}

	os.MkdirAll(
		filepath.Join(
			path,
			"keps",
			opts.SIG,
			opts.Name,
		),
		os.ModePerm,
	)
	newPath := filepath.Join(path, "keps", opts.SIG, opts.Name, "kep.yaml")
	fmt.Fprintf(c.Out, "writing KEP to %s\n", newPath)
	return ioutil.WriteFile(newPath, b, os.ModePerm)
}
