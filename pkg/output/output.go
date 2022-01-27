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

package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"

	"k8s.io/enhancements/api"
	"k8s.io/enhancements/pkg/yaml"
)

const (
	DefaultFormat = "table"
)

// ValidFormats returns the list of valid formats for NewOutput
func ValidFormats() []string {
	return []string{"csv", "json", "table", "yaml"}
}

// Output is capable of printing... well pretty much just KEPs, but if there
// is interest in printing other types in api, this could be the place
type Output interface {
	PrintProposals([]*api.Proposal)
}

// output holds out/err streams for Output implementations
type output struct {
	Out io.Writer
	Err io.Writer
}

// columnOutput holds PrintConfigs in addition to out/err streams for column
// oriented Output implementations (e.g. CSVOutput)
type columnOutput struct {
	output
	Configs []PrintConfig
}

// NewOutput returns an Output of the given format that will write to the given
// out and err Writers, or return an err if the format isn't one of ValidFormats()
func NewOutput(format string, out, err io.Writer) (Output, error) {
	switch format {
	case "json":
		return &JSONOutput{Out: out, Err: err}, nil
	case "yaml":
		return &YAMLOutput{Out: out, Err: err}, nil
	case "csv":
		return &CSVOutput{
			output: output{
				Out: out,
				Err: err,
			},
			Configs: DefaultPrintConfigs(
				"Title",
				"Authors",
				"SIG",
				"Stage",
				"Status",
				"LastUpdated",
				"Link",
			),
		}, nil
	case "table":
		return &TableOutput{
			output: output{
				Out: out,
				Err: err,
			},
			Configs: DefaultPrintConfigs(
				"LastUpdated",
				"Stage",
				"Status",
				"SIG",
				"Authors",
				"Title",
				"Link",
			),
		}, nil
	}
	return nil, fmt.Errorf("unsupported output format: %s. Valid values: %s", format, ValidFormats())
}

type YAMLOutput output

// PrintProposals prints KEPs array in YAML format
func (o *YAMLOutput) PrintProposals(proposals []*api.Proposal) {
	data, err := yaml.Marshal(proposals)
	if err != nil {
		fmt.Fprintf(o.Err, "error printing KEPs as YAML: %s", err)
		return
	}
	fmt.Fprintln(o.Out, string(data))
}

type JSONOutput output

// PrintProposals prints KEPs array in JSON format
func (o *JSONOutput) PrintProposals(proposals []*api.Proposal) {
	data, err := json.Marshal(proposals)
	if err != nil {
		fmt.Fprintf(o.Err, "error printing KEPs as JSON: %s", err)
		return
	}

	fmt.Fprintln(o.Out, string(data))
}

type TableOutput columnOutput

// PrintProposals prints KEPs array in table format
func (o *TableOutput) PrintProposals(proposals []*api.Proposal) {
	if len(o.Configs) == 0 {
		return
	}

	table := tablewriter.NewWriter(o.Out)

	headers := make([]string, 0, len(o.Configs))
	for _, c := range o.Configs {
		headers = append(headers, c.Title())
	}

	table.SetHeader(headers)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, k := range proposals {
		var s []string
		for _, c := range o.Configs {
			s = append(s, c.Value(k))
		}
		table.Append(s)
	}

	table.Render()
}

type CSVOutput columnOutput

// PrintProposals prins KEPs array in CSV format
func (o *CSVOutput) PrintProposals(proposals []*api.Proposal) {
	w := csv.NewWriter(o.Out)
	defer w.Flush()

	headers := make([]string, 0, len(o.Configs))
	for _, c := range o.Configs {
		headers = append(headers, c.Title())
	}
	if err := w.Write(headers); err != nil {
		fmt.Fprintf(o.Err, "error printing keps as CSV: %s", err)
	}

	for _, p := range proposals {
		var row []string
		for _, c := range o.Configs {
			row = append(row, c.Value(p))
		}
		if err := w.Write(row); err != nil {
			fmt.Fprintf(o.Err, "error printing keps as CSV: %s", err)
		}
	}
}

// PrintConfig defines how a given Proposal field should be formatted for
// columnar output (e.g. TableOutput, CSVOutput)
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

func DefaultPrintConfigs(names ...string) []PrintConfig {
	// TODO: Refactor out anonymous funcs
	defaultConfig := map[string]printConfig{
		"Authors":     {"Authors", func(k *api.Proposal) string { return strings.Join(k.Authors, ", ") }},
		"LastUpdated": {"Updated", func(k *api.Proposal) string { return k.LastUpdated }},
		"SIG": {"SIG", func(k *api.Proposal) string {
			if strings.HasPrefix(k.OwningSIG, "sig-") {
				return k.OwningSIG[4:]
			}

			return k.OwningSIG
		}},
		"Stage":  {"Stage", func(k *api.Proposal) string { return string(k.Stage) }},
		"Status": {"Status", func(k *api.Proposal) string { return string(k.Status) }},
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
	configs := make([]PrintConfig, 0, 10)
	for _, n := range names {
		// copy to allow it to be tweaked by the caller
		c := defaultConfig[n]
		configs = append(configs, &c)
	}
	return configs
}
