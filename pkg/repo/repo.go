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
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/go-github/v33/github"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/kepval"
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
	ProposalReadme  string

	// Auth
	TokenPath string
	Token     string

	// Document handlers
	KEPHandler *api.KEPHandler
	PRRHandler *api.PRRHandler

	// Templates
	ProposalTemplate []byte
}

// New returns a new repo client configured to use the the normal os.Stdxxx and Filesystem
func New(repoPath string) (*Repo, error) {
	fetcher := api.DefaultGroupFetcher()
	return NewRepo(repoPath, fetcher)
}

func NewRepo(repoPath string, fetcher api.GroupFetcher) (*Repo, error) {
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

	proposalReadme := filepath.Join(proposalPath, "README.md")
	fi, err = os.Stat(proposalReadme)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"getting file info for proposal README path %s",
			proposalPath,
		)
	}

	if !fi.Mode().IsRegular() {
		return nil, errors.Wrap(
			err,
			"checking if proposal README is a file",
		)
	}

	groups, err := fetcher.FetchGroups()
	if err != nil {
		return nil, fmt.Errorf("fetching groups: %w", err)
	}

	prrApprovers, err := fetcher.FetchPRRApprovers()
	if err != nil {
		return nil, fmt.Errorf("fetching PRR approvers: %w", err)
	}

	kepHandler := &api.KEPHandler{Groups: groups, PRRApprovers: prrApprovers}

	prrHandler := &api.PRRHandler{PRRApprovers: prrApprovers}

	repo := &Repo{
		BasePath:        repoPath,
		ProposalPath:    proposalPath,
		PRRApprovalPath: prrApprovalPath,
		ProposalReadme:  proposalReadme,
		KEPHandler:      kepHandler,
		PRRHandler:      prrHandler,
	}

	proposalTemplate, err := repo.getProposalTemplate()
	if err != nil {
		return nil, errors.Wrap(err, "getting proposal template")
	}

	repo.ProposalTemplate = proposalTemplate

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
			logrus.Debugf("processing filename %s", info.Name())

			if err != nil {
				return err
			}

			// true if the file is a symlink
			if info.Mode()&os.ModeSymlink != 0 {
				// Assume symlink from old KEP location to new. The new location
				// will be processed separately, so no need to process it here.
				logrus.Debugf("%s is a symlink", info.Name())
				return nil
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			if info.Name() == ProposalMetadataFilename {
				logrus.Debugf("adding %s as KEP metadata", info.Name())
				keps = append(keps, path)
				return filepath.SkipDir
			}

			if info.Name() == ProposalFilename {
				return nil
			}

			logrus.Debugf("adding %s as KEP metadata", info.Name())
			keps = append(keps, path)
			return nil
		},
	)

	return keps, err
}

func (r *Repo) LoadLocalKEPs(sig string) ([]*api.Proposal, error) {
	// KEPs in the local filesystem
	files, err := r.findLocalKEPMeta(sig)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"searching for local KEPs from %s",
			sig,
		)
	}

	logrus.Debugf("loading the following local KEPs: %v", files)

	var allKEPs []*api.Proposal
	for _, k := range files {
		if filepath.Ext(k) == ".yaml" {
			kep, err := r.loadKEPFromYaml(k)
			if err != nil {
				return nil, errors.Wrapf(
					err,
					"reading KEP %s from yaml",
					k,
				)
			}

			allKEPs = append(allKEPs, kep)
		}
	}

	logrus.Debugf("returning %d local KEPs", len(allKEPs))

	return allKEPs, nil
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
				logrus.Warnf("error reading KEP %v: %v", k, err)
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
	if err != nil {
		return nil, errors.Wrapf(err, "getting file info for %s", kepPath)
	}

	return r.loadKEPFromYaml(kepPath)
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
	handler := r.PRRHandler
	err = kepval.ValidatePRR(&p, handler, r.PRRApprovalPath)
	if err != nil {
		logrus.Errorf(
			"%v",
			errors.Wrapf(err, "validating PRR for %s", p.Name),
		)
	} else {
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

		approval, err := handler.Parse(prrFile)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing PRR")
		}

		approver, err := approval.ApproverForStage(p.Stage)
		if err != nil {
			logrus.Errorf(
				"%v",
				errors.Wrapf(err, "getting PRR approver for %s stage", p.Stage),
			)
		}

		for _, a := range p.PRRApprovers {
			if a == approver {
				approver = ""
			}
		}

		if approver != "" {
			p.PRRApprovers = append(p.PRRApprovers, approver)
		}
	}

	return &p, nil
}
