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

package repo

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/output"
)

type QueryOpts struct {
	Groups      []string
	Status      []string
	Stage       []string
	PRRApprover []string
	Author      []string
	Approver    []string
	IncludePRs  bool
	Output      string
}

// Validate returns an error if any QueryOpts fields are invalid
func (o *QueryOpts) Validate() error {
	if !sliceContains(output.ValidFormats(), o.Output) {
		return fmt.Errorf("unsupported output format: %s. Valid values: %v", o.Output, output.ValidFormats())
	}

	// TODO: check the valid values of stage, status, etc.
	return nil
}

func (r *Repo) PrepareQueryOpts(opts *QueryOpts) error {
	err := opts.Validate()
	if err != nil {
		return fmt.Errorf("invalid query opts: %w", err)
	}

	groups := r.KEPHandler.Groups

	if len(opts.Groups) > 0 {
		sigs, err := selectByRegexp(groups, opts.Groups)
		if err != nil {
			return err
		}

		if len(sigs) == 0 {
			return fmt.Errorf("no SIGs match any of: %v", opts.Groups)
		}

		opts.Groups = sigs
	} else {
		// if no SIGs are passed, list KEPs from all SIGs
		opts.Groups = groups
	}
	return nil
}

// Query searches the local repo and possibly GitHub for KEPs
// that match the search criteria.
func (r *Repo) Query(opts *QueryOpts) ([]*api.Proposal, error) {
	logrus.Info("Searching for KEPs...")

	err := r.PrepareQueryOpts(opts)
	if err != nil {
		return nil, fmt.Errorf("invalid query options: %w", err)
	}

	if r.TokenPath != "" {
		logrus.Infof("Setting GitHub token: %v", r.TokenPath)
		if tokenErr := r.SetGitHubToken(r.TokenPath); tokenErr != nil {
			return nil, errors.Wrapf(tokenErr, "setting GitHub token")
		}
	}

	allKEPs := make([]*api.Proposal, 0, 10)
	// load the KEPs for each listed SIG
	for _, sig := range opts.Groups {
		// KEPs in the local filesystem
		localKEPs, err := r.LoadLocalKEPs(sig)
		if err != nil {
			return nil, errors.Wrap(err, "loading local KEPs")
		}

		allKEPs = append(allKEPs, localKEPs...)

		// Open PRs; existing KEPs with open PRs will be shown twice
		if opts.IncludePRs {
			prKeps, err := r.loadKEPPullRequests(sig)
			if err != nil {
				logrus.Warnf("error searching for KEP PRs from %s: %s", sig, err)
			}
			if prKeps != nil {
				allKEPs = append(allKEPs, prKeps...)
			}
		}
	}

	logrus.Debugf("all KEPs collected: %+v", allKEPs)

	// filter the KEPs by criteria
	allowedStatus := sliceToMap(opts.Status)
	allowedStage := sliceToMap(opts.Stage)
	allowedPRR := sliceToMap(opts.PRRApprover)
	allowedAuthor := sliceToMap(opts.Author)
	allowedApprover := sliceToMap(opts.Approver)

	results := make([]*api.Proposal, 0, 10)
	for _, k := range allKEPs {
		if k == nil {
			return nil, errors.New("one of the KEPs in query was nil")
		}

		logrus.Debugf("current KEP: %v", k)

		if len(opts.Status) > 0 && !allowedStatus[k.Status] {
			continue
		}
		if len(opts.Stage) > 0 && !allowedStage[k.Stage] {
			continue
		}
		if len(opts.PRRApprover) > 0 && !atLeastOne(k.PRRApprovers, allowedPRR) {
			continue
		}
		if len(opts.Author) > 0 && !atLeastOne(k.Authors, allowedAuthor) {
			continue
		}
		if len(opts.Approver) > 0 && !atLeastOne(k.Approvers, allowedApprover) {
			continue
		}

		results = append(results, k)
	}

	return results, nil
}

func sliceToMap(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}

func sliceContains(s []string, e string) bool {
	for _, k := range s {
		if k == e {
			return true
		}
	}

	return false
}

// returns all strings in vals that match at least one
// regexp in regexps
func selectByRegexp(vals, regexps []string) ([]string, error) {
	var matches []string
	for _, s := range vals {
		for _, r := range regexps {
			found, err := regexp.MatchString(r, s)
			if err != nil {
				return matches, err
			}
			if found {
				matches = append(matches, s)
				break
			}
		}
	}

	return matches, nil
}

// returns true if at least one of vals is in the allowed map
func atLeastOne(vals []string, allowed map[string]bool) bool {
	for _, v := range vals {
		if allowed[v] {
			return true
		}
	}

	return false
}
