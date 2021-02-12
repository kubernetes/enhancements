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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v32/github"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"

	"k8s.io/enhancements/api"
	"k8s.io/test-infra/prow/git"
)

type CommonArgs struct {
	// command options
	LogLevel  string
	RepoPath  string // override the default settings
	TokenPath string

	// KEP options
	KEP    string // KEP name sig-xxx/xxx-name
	Name   string
	Number string
	SIG    string
}

func (c *CommonArgs) validateAndPopulateKEP(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("must provide a path for the new KEP like sig-architecture/0000-new-kep")
	}
	if len(args) == 1 {
		kep := args[0]
		re := regexp.MustCompile(`([a-z\\-]+)/((\\d+)-.+)`)
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
	Token    string
	In       io.Reader
	Out      io.Writer
	Err      io.Writer
}

func (c *Client) findEnhancementsRepo(opts *CommonArgs) (string, error) {
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

func (c *Client) SetGitHubToken(opts *CommonArgs) error {
	if opts.TokenPath != "" {
		token, err := ioutil.ReadFile(opts.TokenPath)
		if err != nil {
			return err
		}

		c.Token = strings.Trim(string(token), "\n\r")
	}

	return nil
}

// getKepTemplate reads the kep.yaml template from the local
// (per c.RepoPath) k/enhancements, but this could be replaced with a
// template via packr or fetched from Github?
func (c *Client) getKepTemplate(repoPath string) (*api.Proposal, error) {
	var p api.Proposal
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

// TODO: Unused?
func validateKEP(p *api.Proposal) error {
	b, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	r := bytes.NewReader(b)
	parser, err := api.NewKEPHandler()
	if err != nil {
		return errors.Wrap(err, "creating KEP handler")
	}

	kep, err := parser.Parse(r)
	if err != nil {
		return fmt.Errorf("kep is invalid: %s", kep.Error)
	}

	return nil
}

func findLocalKEPMeta(repoPath, sig string) ([]string, error) {
	rootPath := filepath.Join(
		repoPath,
		"keps",
		sig,
	)

	keps := []string{}

	// if the SIG doesn't have a dir, it has no KEPs
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return keps, nil
	}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if info.Name() == "kep.yaml" {
			keps = append(keps, path)
			return filepath.SkipDir
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		if info.Name() == "README.md" {
			return nil
		}
		keps = append(keps, path)
		return nil
	})

	return keps, err
}

func (c *Client) loadLocalKEPs(repoPath, sig string) []*api.Proposal {
	// KEPs in the local filesystem
	files, err := findLocalKEPMeta(repoPath, sig)
	if err != nil {
		fmt.Fprintf(c.Err, "error searching for local KEPs from %s: %s\n", sig, err)
	}

	var allKEPs []*api.Proposal
	for _, k := range files {
		if filepath.Ext(k) == ".yaml" {
			kep, err := c.loadKEPFromYaml(k)
			if err != nil {
				fmt.Fprintf(c.Err, "error reading KEP %s: %s\n", k, err)
			} else {
				allKEPs = append(allKEPs, kep)
			}
		} else {
			kep, err := c.loadKEPFromOldStyle(k)
			if err != nil {
				fmt.Fprintf(c.Err, "error reading KEP %s: %s\n", k, err)
			} else {
				allKEPs = append(allKEPs, kep)
			}
		}
	}
	return allKEPs
}

func (c *Client) loadKEPPullRequests(sig string) ([]*api.Proposal, error) {
	var auth *http.Client
	ctx := context.Background()
	if c.Token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.Token})
		auth = oauth2.NewClient(ctx, ts)
	}

	gh := github.NewClient(auth)
	allPulls := []*github.PullRequest{}
	opt := &github.PullRequestListOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for {
		pulls, resp, err := gh.PullRequests.List(ctx, "kubernetes", "enhancements", opt)
		if err != nil {
			return nil, err
		}
		allPulls = append(allPulls, pulls...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	kepPRs := make([]*github.PullRequest, 10)
	for _, pr := range allPulls {
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

		kepPRs = append(kepPRs, pr)
	}

	if len(kepPRs) == 0 {
		return nil, nil
	}

	// Pull a temporary clone of the repo
	g, err := git.NewClient()
	if err != nil {
		return nil, err
	}

	g.SetCredentials("", func() []byte { return []byte{} })
	g.SetRemote("https://github.com")
	repo, err := g.Clone("kubernetes", "enhancements")
	if err != nil {
		return nil, err
	}

	// read out each PR, and create a Proposal for each KEP that is
	// touched by a PR. This may result in multiple versions of the same KEP.
	var allKEPs []*api.Proposal
	for _, pr := range kepPRs {
		files, _, err := gh.PullRequests.ListFiles(context.Background(), "kubernetes", "enhancements",
			pr.GetNumber(), &github.ListOptions{})
		if err != nil {
			return nil, err
		}

		kepNames := make(map[string]bool, 10)
		for _, file := range files {
			if !strings.HasPrefix(*file.Filename, "keps/"+sig+"/") {
				continue
			}

			kk := strings.Split(*file.Filename, "/")
			if len(kk) < 3 {
				continue
			}

			if strings.HasSuffix(kk[2], ".md") {
				kepNames[kk[2][0:len(kk[2])-3]] = true
			} else {
				kepNames[kk[2]] = true
			}
		}

		if len(kepNames) == 0 {
			continue
		}

		err = repo.CheckoutPullRequest(pr.GetNumber())
		if err != nil {
			return nil, err
		}
		// read all these KEPs
		for k := range kepNames {
			kep, err := c.readKEP(repo.Directory(), sig, k)
			if err != nil {
				fmt.Fprintf(c.Err, "ERROR READING KEP %s: %s\n", k, err)
			} else {
				kep.PRNumber = strconv.Itoa(pr.GetNumber())
				allKEPs = append(allKEPs, kep)
			}
		}
	}

	return allKEPs, nil
}

func (c *Client) readKEP(repoPath, sig, name string) (*api.Proposal, error) {
	kepPath := filepath.Join(
		repoPath,
		"keps",
		sig,
		name,
		"kep.yaml")

	_, err := os.Stat(kepPath)
	if err == nil {
		return c.loadKEPFromYaml(kepPath)
	}

	// No kep.yaml, treat as old-style KEP
	kepPath = filepath.Join(
		repoPath,
		"keps",
		sig,
		name) + ".md"

	return c.loadKEPFromOldStyle(kepPath)
}

func (c *Client) loadKEPFromYaml(kepPath string) (*api.Proposal, error) {
	b, err := ioutil.ReadFile(kepPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read KEP metadata: %s", err)
	}
	var p api.Proposal
	err = yaml.Unmarshal(b, &p)
	if err != nil {
		return nil, fmt.Errorf("unable to load KEP metadata: %s", err)
	}
	p.Name = filepath.Base(filepath.Dir(kepPath))

	// Read the PRR approval file and add any listed PRR approvers in there
	// to the PRR approvers list in the KEP. this is a hack while we transition
	// away from PRR approvers listed in kep.yaml
	prrPath := filepath.Dir(kepPath)
	prrPath = filepath.Dir(prrPath)
	sig := filepath.Base(prrPath)
	prrPath = filepath.Join(
		filepath.Dir(prrPath),
		"prod-readiness",
		sig,
		p.Number+".yaml",
	)

	prrFile, err := os.Open(prrPath)
	if os.IsNotExist(err) {
		return &p, nil
	}

	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %v\n", prrPath, err)
	}

	parser, err := api.NewPRRHandler()
	if err != nil {
		return nil, errors.Wrap(err, "creating new PRR handler")
	}

	approval, err := parser.Parse(prrFile)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing PRR")
	}

	if approval.Error != nil {
		fmt.Fprintf(c.Err, "WARNING: could not parse prod readiness request for KEP %s: %s\n", p.Number, approval.Error)
	}

	approver, err := approval.ApproverForStage(p.Stage)
	if err != nil {
		return nil, errors.Wrapf(err, "getting PRR approver for %s stage", p.Stage)
	}

	for _, a := range p.PRRApprovers {
		if a == approver {
			approver = ""
		}
	}

	if approver != "" {
		p.PRRApprovers = append(p.PRRApprovers, approver)
	}

	return &p, nil
}

func (c *Client) loadKEPFromOldStyle(kepPath string) (*api.Proposal, error) {
	b, err := ioutil.ReadFile(kepPath)
	if err != nil {
		return nil, fmt.Errorf("no kep.yaml, but failed to read as old-style KEP: %s", err)
	}

	r := bytes.NewReader(b)

	parser, err := api.NewKEPHandler()
	if err != nil {
		return nil, errors.Wrap(err, "creating new KEP handler")
	}

	kep, err := parser.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("kep is invalid: %s", kep.Error)
	}

	kep.Name = filepath.Base(kepPath)
	return kep, nil
}

func (c *Client) writeKEP(kep *api.Proposal, opts *CommonArgs) error {
	path, err := c.findEnhancementsRepo(opts)
	if err != nil {
		return fmt.Errorf("unable to write KEP: %s", err)
	}

	b, err := yaml.Marshal(kep)
	if err != nil {
		return fmt.Errorf("KEP is invalid: %s", err)
	}

	if mkErr := os.MkdirAll(
		filepath.Join(
			path,
			"keps",
			opts.SIG,
			opts.Name,
		),
		os.ModePerm,
	); mkErr != nil {
		return errors.Wrapf(mkErr, "creating KEP directory")
	}

	newPath := filepath.Join(path, "keps", opts.SIG, opts.Name, "kep.yaml")
	fmt.Fprintf(c.Out, "writing KEP to %s\n", newPath)

	return ioutil.WriteFile(newPath, b, os.ModePerm)
}

type PrintConfig interface {
	Title() string
	Value(*api.Proposal) string
}

type printConfig struct {
	title     string
	valueFunc func(*api.Proposal) string
}

func (p *printConfig) Title() string { return p.title }
func (p *printConfig) Value(k *api.Proposal) string {
	return p.valueFunc(k)
}

// TODO: Refactor out anonymous funcs
var defaultConfig = map[string]printConfig{
	"Authors":     {"Authors", func(k *api.Proposal) string { return strings.Join(k.Authors, ", ") }},
	"LastUpdated": {"Updated", func(k *api.Proposal) string { return k.LastUpdated }},
	"SIG": {"SIG", func(k *api.Proposal) string {
		if strings.HasPrefix(k.OwningSIG, "sig-") {
			return k.OwningSIG[4:]
		}

		return k.OwningSIG
	}},
	"Stage":  {"Stage", func(k *api.Proposal) string { return k.Stage }},
	"Status": {"Status", func(k *api.Proposal) string { return k.Status }},
	"Title": {"Title", func(k *api.Proposal) string {
		if k.PRNumber == "" {
			return k.Title
		}

		return "PR#" + k.PRNumber + " - " + k.Title
	}},
	"Link": {"Link", func(k *api.Proposal) string {
		if k.PRNumber == "" {
			return "https://git.k8s.io/enhancements/keps/" + k.OwningSIG + "/" + k.Name
		}

		return "https://github.com/kubernetes/enhancements/pull/" + k.PRNumber
	}},
}

func DefaultPrintConfigs(names ...string) []PrintConfig {
	configs := make([]PrintConfig, 10)
	for _, n := range names {
		// copy to allow it to be tweaked by the caller
		c := defaultConfig[n]
		configs = append(configs, &c)
	}

	return configs
}

func (c *Client) PrintTable(configs []PrintConfig, proposals []*api.Proposal) {
	if len(configs) == 0 {
		return
	}

	table := tablewriter.NewWriter(c.Out)

	headers := make([]string, 10)
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

// PrintYAML outputs keps array as YAML to c.Out
func (c *Client) PrintYAML(proposals []*api.Proposal) {
	data, err := yaml.Marshal(proposals)
	if err != nil {
		fmt.Fprintf(c.Err, "error printing keps as YAML: %s", err)
		return
	}

	fmt.Fprintln(c.Out, string(data))
}

// PrintJSON outputs keps array as YAML to c.Out
func (c *Client) PrintJSON(proposals []*api.Proposal) {
	data, err := json.Marshal(proposals)
	if err != nil {
		fmt.Fprintf(c.Err, "error printing keps as JSON: %s", err)
		return
	}

	fmt.Fprintln(c.Out, string(data))
}
