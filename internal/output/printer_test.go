package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ssoriche/kubectl-karpenter/internal/collector"
)

func TestPrintTable(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter("", false)
	p.out = &buf

	pools := []collector.NodePoolInfo{
		{Name: "default", NodeCount: 2, CPUPercent: 50, MemPercent: 75},
		{Name: "gpu", NodeCount: 1, CPUPercent: 100, MemPercent: 25},
	}
	total := collector.NodePoolInfo{Name: "TOTAL", NodeCount: 3, CPUPercent: 66, MemPercent: 58}

	err := p.PrintTable(pools, total)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	// Check header
	if !strings.Contains(output, "NODEPOOL") {
		t.Error("missing NODEPOOL header")
	}
	if !strings.Contains(output, "NODES") {
		t.Error("missing NODES header")
	}

	// Check pool rows
	if !strings.Contains(output, "default") {
		t.Error("missing default pool")
	}
	if !strings.Contains(output, "gpu") {
		t.Error("missing gpu pool")
	}

	// Check total row
	if !strings.Contains(output, "TOTAL") {
		t.Error("missing TOTAL row")
	}

	// Check bars contain block chars
	if !strings.Contains(output, "█") {
		t.Error("missing filled bar chars")
	}
	if !strings.Contains(output, "░") {
		t.Error("missing empty bar chars")
	}
}

func TestPrintTableNoHeaders(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter("", true)
	p.out = &buf

	pools := []collector.NodePoolInfo{
		{Name: "default", NodeCount: 1, CPUPercent: 50, MemPercent: 50},
	}
	total := collector.NodePoolInfo{Name: "TOTAL", NodeCount: 1, CPUPercent: 50, MemPercent: 50}

	err := p.PrintTable(pools, total)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if strings.Contains(output, "NODEPOOL") {
		t.Error("should not contain header when noHeaders=true")
	}
}
