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
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	krgh "k8s.io/release/pkg/github"
	"k8s.io/test-infra/prow/git"

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/kepval"
	"k8s.io/enhancements/pkg/yaml"
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

	// Temporary caches
	// a local git clone of remoteOrg/remoteRepo
	gitRepo *git.Repo
	// all open pull requests for remoteOrg/remoteRepo
	allPRs []*github.PullRequest
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
		return nil, fmt.Errorf("getting file info for proposal path %s: %w", proposalPath, err)
	}

	if !fi.IsDir() {
		return nil, fmt.Errorf("checking if proposal path is a directory: %w", err)
	}

	prrApprovalPath := filepath.Join(proposalPath, PRRApprovalPathStub)
	fi, err = os.Stat(prrApprovalPath)
	if err != nil {
		return nil, fmt.Errorf("getting file info for PRR approval path %s: %w", prrApprovalPath, err)
	}

	if !fi.IsDir() {
		return nil, fmt.Errorf("checking if PRR approval path is a directory: %w", err)
	}

	proposalReadme := filepath.Join(proposalPath, "README.md")
	fi, err = os.Stat(proposalReadme)
	if err != nil {
		return nil, fmt.Errorf("getting file info for proposal README path %s: %w", proposalPath, err)
	}

	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("checking if proposal README is a file: %w", err)
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
		return nil, fmt.Errorf("getting proposal template: %w", err)
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
	sigPath := filepath.Join(r.ProposalPath, sig)

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
				path, err = filepath.Rel(r.BasePath, path)
				if err != nil {
					return err
				}
				keps = append(keps, path)
				return filepath.SkipDir
			}

			if info.Name() == ProposalFilename {
				return nil
			}

			return nil
		},
	)

	return keps, err
}

func (r *Repo) LoadLocalKEPs(sig string) ([]*api.Proposal, error) {
	// KEPs in the local filesystem
	files, err := r.findLocalKEPMeta(sig)
	if err != nil {
		return nil, fmt.Errorf("searching for local KEPs from %s: %w", sig, err)
	}

	logrus.Debugf("loading the following local KEPs: %v", files)

	allKEPs := make([]*api.Proposal, len(files))
	for i, kepYamlPath := range files {
		kep, err := r.loadKEPFromYaml(r.BasePath, kepYamlPath)
		if err != nil {
			return nil, fmt.Errorf("reading KEP %s from yaml: %w", kepYamlPath, err)
		}

		allKEPs[i] = kep
	}

	logrus.Debugf("returning %d local KEPs", len(allKEPs))

	return allKEPs, nil
}

func (r *Repo) LoadLocalKEP(sig, name string) (*api.Proposal, error) {
	kepPath := filepath.Join(
		ProposalPathStub,
		sig,
		name,
		ProposalMetadataFilename,
	)

	_, err := os.Stat(kepPath)
	if err != nil {
		return nil, fmt.Errorf("getting file info for %s: %w", kepPath, err)
	}

	return r.loadKEPFromYaml(r.BasePath, kepPath)
}

func (r *Repo) LoadPullRequestKEPs(sig string) ([]*api.Proposal, error) {
	// Initialize github client
	logrus.Debugf("Initializing github client to load PRs for sig: %v", sig)
	var auth *http.Client
	ctx := context.Background()
	if r.Token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: r.Token})
		auth = oauth2.NewClient(ctx, ts)
	}
	gh := github.NewClient(auth)

	// Fetch list of all PRs if none exists
	if r.allPRs == nil {
		logrus.Debugf("Initializing list of all PRs for %v/%v", remoteOrg, remoteRepo)
		r.allPRs = []*github.PullRequest{}

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

			r.allPRs = append(r.allPRs, pulls...)
			if resp.NextPage == 0 {
				break
			}

			opt.Page = resp.NextPage
		}
	}

	// Find KEP PRs for the given sig
	kepPRs := []*github.PullRequest{}
	sigLabel := strings.Replace(sig, "-", "/", 1)
	logrus.Debugf("Searching list of %v PRs for %v/%v with labels: [%v, %v]", len(r.allPRs), remoteOrg, remoteRepo, sigLabel, proposalLabel)
	for _, pr := range r.allPRs {
		foundKind, foundSIG := false, false

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

		logrus.Debugf("Found #%v", pr.GetHTMLURL())

		kepPRs = append(kepPRs, pr)
	}
	logrus.Debugf("Found %v PRs for %v/%v with labels: [%v, %v]", len(kepPRs), remoteOrg, remoteRepo, sigLabel, proposalLabel)

	if len(kepPRs) == 0 {
		return nil, nil
	}

	// Pull a temporary clone of the repo if none already exists
	if r.gitRepo == nil {
		g, err := git.NewClient()
		if err != nil {
			return nil, err
		}

		g.SetCredentials("", func() []byte { return []byte{} })
		g.SetRemote(krgh.GitHubURL)

		r.gitRepo, err = g.Clone(remoteOrg, remoteRepo)
		if err != nil {
			return nil, err
		}
	}

	// read out each PR, and create a Proposal for each KEP that is
	// touched by a PR. This may result in multiple versions of the same KEP.
	var allKEPs []*api.Proposal
	for _, pr := range kepPRs {
		logrus.Debugf("Getting list of files for %v", pr.GetHTMLURL())
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

		err = r.gitRepo.CheckoutPullRequest(pr.GetNumber())
		if err != nil {
			return nil, err
		}

		// read all these KEPs
		for k := range kepNames {
			kepPath := filepath.Join(
				ProposalPathStub,
				sig,
				k,
				ProposalMetadataFilename,
			)
			kep, err := r.loadKEPFromYaml(r.gitRepo.Directory(), kepPath)
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

// loadKEPFromYaml will return a Proposal from a kep.yaml at the given kepPath
// within the given repoPath, or an error if the Proposal is invalid
func (r *Repo) loadKEPFromYaml(repoPath, kepPath string) (*api.Proposal, error) {
	fullKEPPath := filepath.Join(repoPath, kepPath)
	b, err := ioutil.ReadFile(fullKEPPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read KEP metadata for %s: %w", fullKEPPath, err)
	}

	var p api.Proposal
	err = yaml.UnmarshalStrict(b, &p)
	if err != nil {
		return nil, fmt.Errorf("unable to load KEP metadata: %s", err)
	}

	p.Name = filepath.Base(filepath.Dir(kepPath))
	prrApprovalPath := filepath.Join(repoPath, ProposalPathStub, PRRApprovalPathStub)

	// Read the PRR approval file and add any listed PRR approvers in there
	// to the PRR approvers list in the KEP. this is a hack while we transition
	// away from PRR approvers listed in kep.yaml
	handler := r.PRRHandler
	err = kepval.ValidatePRR(&p, handler, prrApprovalPath)
	if err != nil {
		logrus.Errorf(
			"%v",
			fmt.Errorf("validating PRR for %s: %w", p.Name, err),
		)
	}

	return &p, nil
}
