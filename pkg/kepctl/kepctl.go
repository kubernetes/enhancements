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
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"

	"k8s.io/enhancements/pkg/kepval/keps"
	"k8s.io/test-infra/prow/git"
)

type CommonArgs struct {
	RepoPath  string //override the default settings
	TokenPath string
	KEP       string //KEP name sig-xxx/xxx-name
	Name      string
	Number    string
	SIG       string
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
	Token    string
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

func (c *Client) SetGitHubToken(opts CommonArgs) error {
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

func findLocalKEPMeta(repoPath, sig string) ([]string, error) {
	rootPath := filepath.Join(
		repoPath,
		"keps",
		sig)

	keps := []string{}

	// if the sig doesn't have a dir, it has no KEPs
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

func (c *Client) loadLocalKEPs(repoPath, sig string) ([]*keps.Proposal) {
	// KEPs in the local filesystem
	files, err := findLocalKEPMeta(repoPath, sig)
	if err != nil {
		fmt.Fprintf(c.Err, "error searching for local KEPs from %s: %s\n", sig, err)
	}

	var allKEPs []*keps.Proposal
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

func (c *Client) loadKEPPullRequests(sig string) ([]*keps.Proposal, error) {
	var auth *http.Client
	ctx := context.Background()
	if c.Token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.Token})
		auth = oauth2.NewClient(ctx, ts)
	}

	gh := github.NewClient(auth)
	pulls, _, err := gh.PullRequests.List(ctx, "kubernetes", "enhancements", &github.PullRequestListOptions{})
	if err != nil {
		return nil, err
	}

	var kepPRs []*github.PullRequest
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
	var allKEPs []*keps.Proposal
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

func (c *Client) readKEP(repoPath string, sig, name string) (*keps.Proposal, error) {
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

func (c *Client) loadKEPFromYaml(kepPath string) (*keps.Proposal, error) {
	b, err := ioutil.ReadFile(kepPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read KEP metadata: %s", err)
	}
	var p keps.Proposal
	err = yaml.Unmarshal(b, &p)
	if err != nil {
		return nil, fmt.Errorf("unable to load KEP metadata: %s", err)
	}
	p.Name = filepath.Base(filepath.Dir(kepPath))
	return &p, nil
}

func (c *Client) loadKEPFromOldStyle(kepPath string) (*keps.Proposal, error) {
	b, err := ioutil.ReadFile(kepPath)
	if err != nil {
		return nil, fmt.Errorf("no kep.yaml, but failed to read as old-style KEP: %s", err)
	}
	r := bytes.NewReader(b)
	parser := &keps.Parser{}

	kep := parser.Parse(r)
	if kep.Error != nil {
		return nil, fmt.Errorf("kep is invalid: %s", kep.Error)
	}
	kep.Name = filepath.Base(kepPath)
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
	return ioutil.WriteFile(newPath, b, 0644)
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
	"Title": {"Title", func(k *keps.Proposal) string {
		if k.PRNumber == "" {
			return k.Title
		} else {
			return "PR#" + k.PRNumber + " - " + k.Title
		}
	}},
	"Link": {"Link", func(k *keps.Proposal) string {
		if k.PRNumber == "" {
			return "https://git.k8s.io/enhancements/keps/" + k.OwningSIG + "/" + k.Name
		} else {
			return "https://github.com/kubernetes/enhancements/pull/" + k.PRNumber
		}
	}},
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

// PrintYAML outputs keps array as YAML to c.Out
func (c *Client) PrintYAML(proposals []*keps.Proposal) {
	data, err := yaml.Marshal(proposals)
	if err != nil {
		fmt.Fprintf(c.Err, "error printing keps as YAML: %s", err)
		return
	}

	fmt.Fprintln(c.Out, string(data))
}

// PrintJSON outputs keps array as YAML to c.Out
func (c *Client) PrintJSON(proposals []*keps.Proposal) {
	data, err := json.Marshal(proposals)
	if err != nil {
		fmt.Fprintf(c.Err, "error printing keps as JSON: %s", err)
		return
	}

	fmt.Fprintln(c.Out, string(data))
}
