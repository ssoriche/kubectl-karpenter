package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"

	"github.com/ssoriche/kubectl-karpenter/internal/collector"
	"github.com/ssoriche/kubectl-karpenter/internal/utilization"
)

const barWidth = 14

type Printer struct {
	out          io.Writer
	outputFormat string
	noHeaders    bool
}

func NewPrinter(outputFormat string, noHeaders bool) *Printer {
	return &Printer{
		out:          os.Stdout,
		outputFormat: outputFormat,
		noHeaders:    noHeaders,
	}
}

func (p *Printer) Print(pools []collector.NodePoolInfo, total collector.NodePoolInfo) error {
	switch p.outputFormat {
	case "json":
		return p.printJSON(pools, total)
	case "yaml":
		return p.printYAML(pools, total)
	default:
		return p.PrintTable(pools, total)
	}
}

func (p *Printer) PrintTable(pools []collector.NodePoolInfo, total collector.NodePoolInfo) error {
	w := tabwriter.NewWriter(p.out, 0, 0, 2, ' ', 0)

	if !p.noHeaders {
		_, _ = fmt.Fprintln(w, "NODEPOOL\tNODES\tCPU\tMEM")
	}

	for _, pool := range pools {
		_, _ = fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
			pool.Name,
			pool.NodeCount,
			utilization.RenderBar(pool.CPUPercent, barWidth),
			utilization.RenderBar(pool.MemPercent, barWidth),
		)
	}

	// Separator line
	_, _ = fmt.Fprintln(w, strings.Repeat("─", 70))

	_, _ = fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
		total.Name,
		total.NodeCount,
		utilization.RenderBar(total.CPUPercent, barWidth),
		utilization.RenderBar(total.MemPercent, barWidth),
	)

	return w.Flush()
}

type poolOutput struct {
	Name       string `json:"name" yaml:"name"`
	NodeCount  int    `json:"nodeCount" yaml:"nodeCount"`
	CPUPercent int    `json:"cpuPercent" yaml:"cpuPercent"`
	MemPercent int    `json:"memPercent" yaml:"memPercent"`
}

type tableOutput struct {
	Pools []poolOutput `json:"pools" yaml:"pools"`
	Total poolOutput   `json:"total" yaml:"total"`
}

func toOutput(pools []collector.NodePoolInfo, total collector.NodePoolInfo) tableOutput {
	out := tableOutput{
		Pools: make([]poolOutput, len(pools)),
		Total: poolOutput{Name: total.Name, NodeCount: total.NodeCount, CPUPercent: total.CPUPercent, MemPercent: total.MemPercent},
	}
	for i, p := range pools {
		out.Pools[i] = poolOutput{Name: p.Name, NodeCount: p.NodeCount, CPUPercent: p.CPUPercent, MemPercent: p.MemPercent}
	}
	return out
}

func (p *Printer) printJSON(pools []collector.NodePoolInfo, total collector.NodePoolInfo) error {
	enc := json.NewEncoder(p.out)
	enc.SetIndent("", "  ")
	return enc.Encode(toOutput(pools, total))
}

func (p *Printer) printYAML(pools []collector.NodePoolInfo, total collector.NodePoolInfo) error {
	enc := yaml.NewEncoder(p.out)
	enc.SetIndent(2)
	return enc.Encode(toOutput(pools, total))
}
