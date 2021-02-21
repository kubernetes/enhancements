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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/go-github/v33/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"

	"k8s.io/enhancements/api"
	krgh "k8s.io/release/pkg/github"
	"k8s.io/test-infra/prow/git"
)

const (
	ProposalPathStub         = "keps"
	ProposalTemplatePathStub = "NNNN-kep-template"
	ProposalFilename         = "README.md"
	ProposalMetadataFilename = "kep.yaml"

	PRRApprovalPathStub = "prod-readiness"

	remoteOrg  = "kubernetes"
	remoteRepo = "enhancements"

	proposalLabel = "kind/kep"
)

type Repo struct {
	// Paths
	BasePath        string
	ProposalPath    string
	PRRApprovalPath string
	TokenPath       string

	// Auth
	Token string

	// I/O
	In  io.Reader
	Out io.Writer
	Err io.Writer

	// Document handlers
	KEPHandler *api.KEPHandler
	PRRHandler *api.PRRHandler

	// Templates
	ProposalTemplate []byte
}

// New returns a new repo client configured to use the the normal os.Stdxxx and Filesystem
func New(repoPath string) (*Repo, error) {
	var err error
	if repoPath == "" {
		repoPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("unable to determine enhancements repo path: %s", err)
		}
	}

	proposalPath := filepath.Join(repoPath, ProposalPathStub)
	fi, err := os.Stat(proposalPath)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"getting file info for proposal path %s",
			proposalPath,
		)
	}

	if !fi.IsDir() {
		return nil, errors.Wrap(
			err,
			"checking if proposal path is a directory",
		)
	}

	prrApprovalPath := filepath.Join(proposalPath, PRRApprovalPathStub)
	fi, err = os.Stat(prrApprovalPath)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"getting file info for PRR approval path %s",
			prrApprovalPath,
		)
	}

	if !fi.IsDir() {
		return nil, errors.Wrap(
			err,
			"checking if PRR approval path is a directory",
		)
	}

	repo := &Repo{
		BasePath:        repoPath,
		ProposalPath:    proposalPath,
		PRRApprovalPath: prrApprovalPath,
		In:              os.Stdin,
		Out:             os.Stdout,
		Err:             os.Stderr,
	}

	proposalTemplate, err := repo.getProposalTemplate()
	if err != nil {
		return nil, errors.Wrap(err, "getting proposal template")
	}

	repo.ProposalTemplate = proposalTemplate

	kepHandler, err := api.NewKEPHandler()
	if err != nil {
		return nil, errors.Wrap(err, "creating KEP handler")
	}

	repo.KEPHandler = kepHandler

	prrHandler, err := api.NewPRRHandler()
	if err != nil {
		return nil, errors.Wrap(err, "creating PRR handler")
	}

	repo.PRRHandler = prrHandler

	// build a default client with normal os.Stdxx and Filesystem access. Tests can build their own
	// with appropriate test objects
	return repo, nil
}

func (r *Repo) SetGitHubToken(tokenFile string) error {
	if tokenFile != "" {
		token, err := ioutil.ReadFile(tokenFile)
		if err != nil {
			return err
		}

		r.Token = strings.Trim(string(token), "\n\r")
	}

	return nil
}

// getProposalTemplate reads the KEP template from the local clone of
// kubernetes/enhancements.
func (r *Repo) getProposalTemplate() ([]byte, error) {
	path := filepath.Join(
		r.ProposalPath,
		ProposalTemplatePathStub,
		ProposalFilename,
	)

	return ioutil.ReadFile(path)
}

func (r *Repo) findLocalKEPMeta(sig string) ([]string, error) {
	sigPath := filepath.Join(
		r.ProposalPath,
		sig,
	)

	keps := []string{}

	// if the SIG doesn't have a dir, it has no KEPs
	if _, err := os.Stat(sigPath); os.IsNotExist(err) {
		return keps, nil
	}

	err := filepath.Walk(
		sigPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			if info.Name() == ProposalMetadataFilename {
				keps = append(keps, path)
				return filepath.SkipDir
			}

			if filepath.Ext(path) != ".md" {
				return nil
			}

			if info.Name() == ProposalFilename {
				return nil
			}

			keps = append(keps, path)
			return nil
		},
	)

	return keps, err
}

func (r *Repo) LoadLocalKEPs(sig string) []*api.Proposal {
	// KEPs in the local filesystem
	files, err := r.findLocalKEPMeta(sig)
	if err != nil {
		fmt.Fprintf(r.Err, "error searching for local KEPs from %s: %s\n", sig, err)
	}

	var allKEPs []*api.Proposal
	for _, k := range files {
		if filepath.Ext(k) == ".yaml" {
			kep, err := r.loadKEPFromYaml(k)
			if err != nil {
				fmt.Fprintf(r.Err, "error reading KEP %s: %s\n", k, err)
			} else {
				allKEPs = append(allKEPs, kep)
			}
		} else {
			kep, err := r.loadKEPFromOldStyle(k)
			if err != nil {
				fmt.Fprintf(r.Err, "error reading KEP %s: %s\n", k, err)
			} else {
				allKEPs = append(allKEPs, kep)
			}
		}
	}

	return allKEPs
}

func (r *Repo) loadKEPPullRequests(sig string) ([]*api.Proposal, error) {
	var auth *http.Client
	ctx := context.Background()
	if r.Token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: r.Token})
		auth = oauth2.NewClient(ctx, ts)
	}

	gh := github.NewClient(auth)
	allPulls := []*github.PullRequest{}
	opt := &github.PullRequestListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		pulls, resp, err := gh.PullRequests.List(
			ctx,
			remoteOrg,
			remoteRepo,
			opt,
		)
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
			if *l.Name == proposalLabel {
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
	g.SetRemote(krgh.GitHubURL)

	repo, err := g.Clone(remoteOrg, remoteRepo)
	if err != nil {
		return nil, err
	}

	// read out each PR, and create a Proposal for each KEP that is
	// touched by a PR. This may result in multiple versions of the same KEP.
	var allKEPs []*api.Proposal
	for _, pr := range kepPRs {
		files, _, err := gh.PullRequests.ListFiles(
			context.Background(),
			remoteOrg,
			remoteRepo,
			pr.GetNumber(),
			&github.ListOptions{},
		)
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
			kep, err := r.ReadKEP(sig, k)
			if err != nil {
				fmt.Fprintf(r.Err, "ERROR READING KEP %s: %s\n", k, err)
			} else {
				kep.PRNumber = strconv.Itoa(pr.GetNumber())
				allKEPs = append(allKEPs, kep)
			}
		}
	}

	return allKEPs, nil
}

func (r *Repo) ReadKEP(sig, name string) (*api.Proposal, error) {
	kepPath := filepath.Join(
		r.ProposalPath,
		sig,
		name,
		ProposalMetadataFilename,
	)

	_, err := os.Stat(kepPath)
	if err == nil {
		return r.loadKEPFromYaml(kepPath)
	}

	// No kep.yaml, treat as old-style KEP
	kepPath = filepath.Join(
		r.ProposalPath,
		sig,
		name,
	) + ".md"

	return r.loadKEPFromOldStyle(kepPath)
}

func (r *Repo) loadKEPFromYaml(kepPath string) (*api.Proposal, error) {
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
		PRRApprovalPathStub,
		sig,
		p.Number+".yaml",
	)

	prrFile, err := os.Open(prrPath)
	if os.IsNotExist(err) {
		return &p, nil
	}

	if err != nil {
		return nil, errors.Wrapf(err, "opening PRR approval %s", prrPath)
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
		fmt.Fprintf(
			r.Err,
			"WARNING: could not parse prod readiness request for KEP %s: %s\n",
			p.Number,
			approval.Error,
		)
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

func (r *Repo) loadKEPFromOldStyle(kepPath string) (*api.Proposal, error) {
	b, err := ioutil.ReadFile(kepPath)
	if err != nil {
		return nil, fmt.Errorf("no kep.yaml, but failed to read as old-style KEP: %s", err)
	}

	reader := bytes.NewReader(b)

	parser, err := api.NewKEPHandler()
	if err != nil {
		return nil, errors.Wrap(err, "creating new KEP handler")
	}

	kep, err := parser.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("kep is invalid: %s", kep.Error)
	}

	kep.Name = filepath.Base(kepPath)
	return kep, nil
}

func (r *Repo) WriteKEP(kep *api.Proposal) error {
	b, err := yaml.Marshal(kep)
	if err != nil {
		return fmt.Errorf("KEP is invalid: %s", err)
	}

	if mkErr := os.MkdirAll(
		filepath.Join(
			r.ProposalPath,
			kep.OwningSIG,
			kep.Name,
		),
		os.ModePerm,
	); mkErr != nil {
		return errors.Wrapf(mkErr, "creating KEP directory")
	}

	newPath := filepath.Join(
		r.ProposalPath,
		kep.OwningSIG,
		kep.Name,
		ProposalMetadataFilename,
	)

	fmt.Fprintf(r.Out, "writing KEP to %s\n", newPath)

	return ioutil.WriteFile(newPath, b, os.ModePerm)
}
