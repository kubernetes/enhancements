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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/go-github/v32/github"
	"github.com/olekukonko/tablewriter"
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

func findLocalKEPs(repoPath string, sig string) ([]string, error) {
	rootPath := filepath.Join(
		repoPath,
		"keps",
		sig)

	keps := []string{}
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if info.Name() == "kep.yaml" {
			keps = append(keps, filepath.Base(filepath.Dir(path)))
			return filepath.SkipDir
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		if info.Name() == "README.md" {
			return nil
		}
		keps = append(keps, info.Name()[0:len(info.Name())-3])
		return nil
	})

	return keps, err
}

func (c *Client) findKEPPullRequests(sig string) (*keps.Proposal, error) {
	gh := github.NewClient(nil)
	pulls, _, err := gh.PullRequests.List(context.Background(), "kubernetes", "enhancements", &github.PullRequestListOptions{})
	if err != nil {
		return nil, err
	}

	for _, pr := range pulls {
		foundKind, foundSIG := false, false
		sigLabel := strings.Replace(sig, "-", "/", 1)
		for _, l := range pr.Labels {
			if *l.Name == "kind/kep" {
				foundKind = true
			}
			if *l.Name == sigLabel {
				foundSIG = true
			}
		}
		if !foundKind || !foundSIG {
			continue
		}
	}

	return nil, nil
}

func (c *Client) readKEP(repoPath string, sig, name string) (*keps.Proposal, error) {
	kepPath := filepath.Join(
		repoPath,
		"keps",
		sig,
		name,
		"kep.yaml")
	_, err := os.Stat(kepPath)
	if err == nil {
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

	// No kep.yaml, treat as old-style KEP
	kepPath = filepath.Join(
		repoPath,
		"keps",
		sig,
		name) + ".md"
	b, err := ioutil.ReadFile(kepPath)
	if err != nil {
		return nil, fmt.Errorf("unable to load KEP metadata: %s", err)
	}
	r := bytes.NewReader(b)
	parser := &keps.Parser{}

	kep := parser.Parse(r)
	if kep.Error != nil {
		return nil, fmt.Errorf("kep is invalid: %s", kep.Error)
	}
	return kep, nil
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

type PrintConfig interface {
	Title() string
	Value(*keps.Proposal) string
}

type printConfig struct {
	title     string
	valueFunc func(*keps.Proposal) string
}

func (p *printConfig) Title() string { return p.title }
func (p *printConfig) Value(k *keps.Proposal) string {
	return p.valueFunc(k)
}

var defaultConfig = map[string]printConfig{
	"Authors":     {"Authors", func(k *keps.Proposal) string { return strings.Join(k.Authors, ", ") }},
	"LastUpdated": {"Updated", func(k *keps.Proposal) string { return k.LastUpdated }},
	"SIG": {"SIG", func(k *keps.Proposal) string {
		if strings.HasPrefix(k.OwningSIG, "sig-") {
			return k.OwningSIG[4:]
		} else {
			return k.OwningSIG
		}
	}},
	"Stage":  {"Stage", func(k *keps.Proposal) string { return k.Stage }},
	"Status": {"Status", func(k *keps.Proposal) string { return k.Status }},
	"Title":  {"Title", func(k *keps.Proposal) string { return k.Title }},
}

func DefaultPrintConfigs(names ...string) []PrintConfig {
	var configs []PrintConfig
	for _, n := range names {
		// copy to allow it to be tweaked by the caller
		c := defaultConfig[n]
		configs = append(configs, &c)
	}
	return configs
}

func (c *Client) PrintTable(configs []PrintConfig, proposals []*keps.Proposal) {
	if len(configs) == 0 {
		return
	}

	table := tablewriter.NewWriter(c.Out)

	var headers []string
	for _, c := range configs {
		headers = append(headers, c.Title())
	}
	table.SetHeader(headers)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, k := range proposals {
		var s []string
		for _, c := range configs {
			s = append(s, c.Value(k))
		}
		table.Append(s)
	}
	table.Render()
}
