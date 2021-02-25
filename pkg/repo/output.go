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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"

	"k8s.io/enhancements/api"
)

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
	configs := make([]PrintConfig, 0, 10)
	for _, n := range names {
		// copy to allow it to be tweaked by the caller
		c := defaultConfig[n]
		configs = append(configs, &c)
	}

	return configs
}

// PrintTable outputs KEPs array as a table
func (r *Repo) PrintTable(configs []PrintConfig, proposals []*api.Proposal) {
	if len(configs) == 0 {
		return
	}

	table := tablewriter.NewWriter(r.Out)

	headers := make([]string, 0, 10)
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

// PrintYAML outputs KEPs array as YAML
func (r *Repo) PrintYAML(proposals []*api.Proposal) {
	data, err := yaml.Marshal(proposals)
	if err != nil {
		fmt.Fprintf(r.Err, "error printing KEPs as YAML: %s", err)
		return
	}

	fmt.Fprintln(r.Out, string(data))
}

// PrintJSON outputs KEPs array as JSON
func (r *Repo) PrintJSON(proposals []*api.Proposal) {
	data, err := json.Marshal(proposals)
	if err != nil {
		fmt.Fprintf(r.Err, "error printing KEPs as JSON: %s", err)
		return
	}

	fmt.Fprintln(r.Out, string(data))
}
