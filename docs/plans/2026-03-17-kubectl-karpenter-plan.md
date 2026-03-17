# kubectl-karpenter Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a kubectl plugin that displays Karpenter NodePool resource utilization with ASCII bar charts.

**Architecture:** Single-command Cobra CLI. Batch-fetches nodes and pods via client-go, groups by NodePool label, aggregates CPU/MEM request utilization, renders tabwriter table with ASCII bars. Follows kubectl-consolidation patterns exactly.

**Tech Stack:** Go 1.24, Cobra, client-go, text/tabwriter, gopkg.in/yaml.v3

**Reference project:** `/Volumes/ziprecruiter/non-zip/kubectl-consolidation` — follow its patterns for project structure, kube client, karpenter version detection, Makefile, goreleaser, devbox, and testing style.

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `devbox.json`
- Create: `Makefile`
- Create: `.gitignore`
- Create: `.golangci.yml`
- Create: `.goreleaser.yaml`
- Create: `cmd/kubectl-karpenter/main.go` (minimal, just prints "hello")

**Step 1: Initialize git repo**

Run: `cd /Volumes/ziprecruiter/non-zip/k9s-karpenter && git init`

**Step 2: Create go.mod**

```go
module github.com/ssoriche/kubectl-karpenter

go 1.24

require (
    github.com/spf13/cobra v1.10.2
    gopkg.in/yaml.v3 v3.0.1
    k8s.io/api v0.32.0
    k8s.io/apimachinery v0.32.0
    k8s.io/client-go v0.32.0
)
```

**Step 3: Create devbox.json**

```json
{
  "$schema": "https://raw.githubusercontent.com/jetify-com/devbox/0.13.1/.schema/devbox.schema.json",
  "packages": [
    "go@1.23",
    "golangci-lint@latest",
    "goreleaser@latest",
    "kubectl@latest"
  ],
  "shell": {
    "init_hook": [
      "echo 'kubectl-karpenter development environment ready'",
      "echo 'Go version:' $(go version)"
    ],
    "scripts": {
      "build": "make build",
      "test": "make test",
      "lint": "make lint",
      "check": "make check"
    }
  }
}
```

**Step 4: Create Makefile**

Identical to kubectl-consolidation's Makefile but with `BINARY_NAME := kubectl-karpenter` and build path `./cmd/kubectl-karpenter`.

**Step 5: Create .gitignore**

```
bin/
coverage.out
coverage.html
.devbox/
.worktrees/
```

**Step 6: Create .golangci.yml**

```yaml
version: "2"

run:
  timeout: 5m
```

**Step 7: Create .goreleaser.yaml**

Same structure as kubectl-consolidation, but with project name `kubectl-karpenter`, binary `kubectl-karpenter`, and main `./cmd/kubectl-karpenter`. GitHub release owner/name `ssoriche/kubectl-karpenter`. Krew install name `karpenter`.

**Step 8: Create minimal main.go**

```go
package main

import (
    "fmt"
    "os"
)

var version = "dev"

func main() {
    fmt.Println("kubectl-karpenter", version)
    os.Exit(0)
}
```

**Step 9: Run go mod tidy, verify build**

Run: `cd /Volumes/ziprecruiter/non-zip/k9s-karpenter && go mod tidy && make build`
Expected: binary at `bin/kubectl-karpenter`

Run: `bin/kubectl-karpenter`
Expected: `kubectl-karpenter dev`

**Step 10: Commit**

```bash
git add -A
git commit -m "feat: scaffold project with build tooling"
```

---

### Task 2: Kube Client Package

**Files:**
- Create: `internal/kube/client.go`

**Step 1: Write the test**

No test needed — this is a thin wrapper around client-go (same as kubectl-consolidation). Testing requires a real cluster.

**Step 2: Write internal/kube/client.go**

Exact same pattern as `/Volumes/ziprecruiter/non-zip/kubectl-consolidation/internal/kube/client.go`:

```go
package kube

import (
    "k8s.io/client-go/discovery"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

func NewClient() (*kubernetes.Clientset, error) {
    config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
        clientcmd.NewDefaultClientConfigLoadingRules(),
        &clientcmd.ConfigOverrides{},
    ).ClientConfig()
    if err != nil {
        return nil, err
    }
    return kubernetes.NewForConfig(config)
}

func NewDiscoveryClient() (discovery.DiscoveryInterface, error) {
    config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
        clientcmd.NewDefaultClientConfigLoadingRules(),
        &clientcmd.ConfigOverrides{},
    ).ClientConfig()
    if err != nil {
        return nil, err
    }
    return discovery.NewDiscoveryClientForConfig(config)
}
```

**Step 3: Verify it compiles**

Run: `go build ./internal/kube/`
Expected: success

**Step 4: Commit**

```bash
git add internal/kube/
git commit -m "feat: add kubernetes client factory"
```

---

### Task 3: Karpenter Version Detection Package

**Files:**
- Create: `internal/karpenter/labels.go`
- Create: `internal/karpenter/version.go`
- Create: `internal/karpenter/crds.go`

**Step 1: Write internal/karpenter/labels.go**

Same constants as kubectl-consolidation's `labels.go` but only the labels we need (no blocker annotations):

```go
package karpenter

const (
    LabelProvisionerName = "karpenter.sh/provisioner-name"
    LabelNodePool        = "karpenter.sh/nodepool"
)
```

**Step 2: Write internal/karpenter/version.go**

Same as kubectl-consolidation's `version.go`: `APIVersion` type, `ClusterCapabilities` struct, `DetectNodeVersion()`, `GetPoolName()`, `DeterminePoolColumnHeader()`.

**Step 3: Write internal/karpenter/crds.go**

Same as kubectl-consolidation's `crds.go`: `DetectCapabilities()`, `HasKarpenter()`.

**Step 4: Verify it compiles**

Run: `go build ./internal/karpenter/`
Expected: success

**Step 5: Commit**

```bash
git add internal/karpenter/
git commit -m "feat: add karpenter version detection"
```

---

### Task 4: Utilization Calculation

**Files:**
- Create: `internal/utilization/utilization.go`
- Create: `internal/utilization/utilization_test.go`

**Step 1: Write the failing test**

Table-driven tests in `internal/utilization/utilization_test.go`. Test `CalculateNodeUtilization` which takes a node and pods, returns cpuRequests, cpuAllocatable, memRequests, memAllocatable as `resource.Quantity` values. Also test `RenderBar` which takes a percent (int) and width (int), returns a string like `[████░░░░░░] 42%`.

```go
package utilization

import (
    "testing"

    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
)

func TestCalculateNodeUtilization(t *testing.T) {
    tests := []struct {
        name        string
        node        *corev1.Node
        pods        []corev1.Pod
        expectedCPU int
        expectedMem int
    }{
        {
            name: "empty node",
            node: &corev1.Node{
                Status: corev1.NodeStatus{
                    Allocatable: corev1.ResourceList{
                        corev1.ResourceCPU:    resource.MustParse("4"),
                        corev1.ResourceMemory: resource.MustParse("8Gi"),
                    },
                },
            },
            pods:        nil,
            expectedCPU: 0,
            expectedMem: 0,
        },
        {
            name: "50% utilization",
            node: &corev1.Node{
                Status: corev1.NodeStatus{
                    Allocatable: corev1.ResourceList{
                        corev1.ResourceCPU:    resource.MustParse("4"),
                        corev1.ResourceMemory: resource.MustParse("8Gi"),
                    },
                },
            },
            pods: []corev1.Pod{
                {
                    Status: corev1.PodStatus{Phase: corev1.PodRunning},
                    Spec: corev1.PodSpec{
                        Containers: []corev1.Container{{
                            Resources: corev1.ResourceRequirements{
                                Requests: corev1.ResourceList{
                                    corev1.ResourceCPU:    resource.MustParse("2"),
                                    corev1.ResourceMemory: resource.MustParse("4Gi"),
                                },
                            },
                        }},
                    },
                },
            },
            expectedCPU: 50,
            expectedMem: 50,
        },
        {
            name: "skip completed pods",
            node: &corev1.Node{
                Status: corev1.NodeStatus{
                    Allocatable: corev1.ResourceList{
                        corev1.ResourceCPU:    resource.MustParse("4"),
                        corev1.ResourceMemory: resource.MustParse("8Gi"),
                    },
                },
            },
            pods: []corev1.Pod{
                {
                    Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
                    Spec: corev1.PodSpec{
                        Containers: []corev1.Container{{
                            Resources: corev1.ResourceRequirements{
                                Requests: corev1.ResourceList{
                                    corev1.ResourceCPU:    resource.MustParse("2"),
                                    corev1.ResourceMemory: resource.MustParse("4Gi"),
                                },
                            },
                        }},
                    },
                },
            },
            expectedCPU: 0,
            expectedMem: 0,
        },
        {
            name: "nil allocatable",
            node: &corev1.Node{
                Status: corev1.NodeStatus{},
            },
            pods:        nil,
            expectedCPU: 0,
            expectedMem: 0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cpu, mem := CalculateNodeUtilization(tt.node, tt.pods)
            if cpu != tt.expectedCPU {
                t.Errorf("CPU = %d, want %d", cpu, tt.expectedCPU)
            }
            if mem != tt.expectedMem {
                t.Errorf("Mem = %d, want %d", mem, tt.expectedMem)
            }
        })
    }
}

func TestRenderBar(t *testing.T) {
    tests := []struct {
        percent  int
        width    int
        expected string
    }{
        {0, 14, "[░░░░░░░░░░░░░░]   0%"},
        {50, 14, "[███████░░░░░░░]  50%"},
        {100, 14, "[██████████████] 100%"},
        {71, 14, "[█████████░░░░░]  71%"},
    }

    for _, tt := range tests {
        t.Run(tt.expected, func(t *testing.T) {
            got := RenderBar(tt.percent, tt.width)
            if got != tt.expected {
                t.Errorf("RenderBar(%d, %d) = %q, want %q", tt.percent, tt.width, got, tt.expected)
            }
        })
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/utilization/`
Expected: compilation error (package doesn't exist yet)

**Step 3: Write internal/utilization/utilization.go**

```go
package utilization

import (
    "fmt"
    "strings"

    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
)

// CalculateNodeUtilization computes CPU and memory request percentage for a node.
func CalculateNodeUtilization(node *corev1.Node, pods []corev1.Pod) (cpuPercent, memPercent int) {
    allocatable := node.Status.Allocatable
    if allocatable == nil {
        return 0, 0
    }

    allocatableCPU := allocatable.Cpu()
    allocatableMem := allocatable.Memory()
    if allocatableCPU.IsZero() || allocatableMem.IsZero() {
        return 0, 0
    }

    var totalCPU, totalMem resource.Quantity
    for _, pod := range pods {
        if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
            continue
        }
        for _, c := range pod.Spec.Containers {
            if c.Resources.Requests != nil {
                if cpu, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
                    totalCPU.Add(cpu)
                }
                if mem, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
                    totalMem.Add(mem)
                }
            }
        }
        for _, c := range pod.Spec.InitContainers {
            if c.Resources.Requests != nil {
                if cpu, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
                    totalCPU.Add(cpu)
                }
                if mem, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
                    totalMem.Add(mem)
                }
            }
        }
    }

    cpuPercent = percentage(totalCPU, *allocatableCPU)
    memPercent = percentage(totalMem, *allocatableMem)
    return cpuPercent, memPercent
}

func percentage(used, total resource.Quantity) int {
    if total.IsZero() {
        return 0
    }
    return int((used.MilliValue() * 100) / total.MilliValue())
}

// RenderBar renders an ASCII utilization bar like [████░░░░░░] 55%
func RenderBar(percent, width int) string {
    filled := (percent * width) / 100
    if filled > width {
        filled = width
    }
    empty := width - filled
    return fmt.Sprintf("[%s%s] %3d%%", strings.Repeat("█", filled), strings.Repeat("░", empty), percent)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test -v ./internal/utilization/`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/utilization/
git commit -m "feat: add utilization calculation and bar rendering"
```

---

### Task 5: Collector — Fetch Nodes & Pods, Aggregate per NodePool

**Files:**
- Create: `internal/collector/collector.go`
- Create: `internal/collector/collector_test.go`

**Step 1: Write the failing test**

Test the aggregation logic: given a list of nodes (with labels) and pods (assigned to nodes), `Aggregate` should return `[]NodePoolInfo` with correct node counts and utilization percentages. Use fake data, no real Kubernetes client needed for this test.

```go
package collector

import (
    "testing"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/api/resource"
)

func TestAggregate(t *testing.T) {
    nodes := []corev1.Node{
        {
            ObjectMeta: metav1.ObjectMeta{
                Name:   "node-1",
                Labels: map[string]string{"karpenter.sh/nodepool": "default"},
            },
            Status: corev1.NodeStatus{
                Allocatable: corev1.ResourceList{
                    corev1.ResourceCPU:    resource.MustParse("4"),
                    corev1.ResourceMemory: resource.MustParse("8Gi"),
                },
            },
        },
        {
            ObjectMeta: metav1.ObjectMeta{
                Name:   "node-2",
                Labels: map[string]string{"karpenter.sh/nodepool": "default"},
            },
            Status: corev1.NodeStatus{
                Allocatable: corev1.ResourceList{
                    corev1.ResourceCPU:    resource.MustParse("4"),
                    corev1.ResourceMemory: resource.MustParse("8Gi"),
                },
            },
        },
        {
            ObjectMeta: metav1.ObjectMeta{
                Name:   "node-3",
                Labels: map[string]string{"karpenter.sh/nodepool": "gpu"},
            },
            Status: corev1.NodeStatus{
                Allocatable: corev1.ResourceList{
                    corev1.ResourceCPU:    resource.MustParse("8"),
                    corev1.ResourceMemory: resource.MustParse("16Gi"),
                },
            },
        },
    }

    podsByNode := map[string][]corev1.Pod{
        "node-1": {
            {
                Status: corev1.PodStatus{Phase: corev1.PodRunning},
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{{
                        Resources: corev1.ResourceRequirements{
                            Requests: corev1.ResourceList{
                                corev1.ResourceCPU:    resource.MustParse("2"),
                                corev1.ResourceMemory: resource.MustParse("4Gi"),
                            },
                        },
                    }},
                },
            },
        },
        "node-2": {
            {
                Status: corev1.PodStatus{Phase: corev1.PodRunning},
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{{
                        Resources: corev1.ResourceRequirements{
                            Requests: corev1.ResourceList{
                                corev1.ResourceCPU:    resource.MustParse("2"),
                                corev1.ResourceMemory: resource.MustParse("4Gi"),
                            },
                        },
                    }},
                },
            },
        },
        "node-3": {
            {
                Status: corev1.PodStatus{Phase: corev1.PodRunning},
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{{
                        Resources: corev1.ResourceRequirements{
                            Requests: corev1.ResourceList{
                                corev1.ResourceCPU:    resource.MustParse("6"),
                                corev1.ResourceMemory: resource.MustParse("12Gi"),
                            },
                        },
                    }},
                },
            },
        },
    }

    results := Aggregate(nodes, podsByNode)

    if len(results) != 2 {
        t.Fatalf("expected 2 pools, got %d", len(results))
    }

    // Results should be sorted by name
    defaultPool := results[0]
    gpuPool := results[1]

    if defaultPool.Name != "default" {
        t.Errorf("expected first pool 'default', got %q", defaultPool.Name)
    }
    if defaultPool.NodeCount != 2 {
        t.Errorf("default pool: expected 2 nodes, got %d", defaultPool.NodeCount)
    }
    // 4/8 CPU = 50%
    if defaultPool.CPUPercent != 50 {
        t.Errorf("default pool: expected CPU 50%%, got %d%%", defaultPool.CPUPercent)
    }
    // 8/16 Gi MEM = 50%
    if defaultPool.MemPercent != 50 {
        t.Errorf("default pool: expected MEM 50%%, got %d%%", defaultPool.MemPercent)
    }

    if gpuPool.Name != "gpu" {
        t.Errorf("expected second pool 'gpu', got %q", gpuPool.Name)
    }
    if gpuPool.NodeCount != 1 {
        t.Errorf("gpu pool: expected 1 node, got %d", gpuPool.NodeCount)
    }
    // 6/8 CPU = 75%
    if gpuPool.CPUPercent != 75 {
        t.Errorf("gpu pool: expected CPU 75%%, got %d%%", gpuPool.CPUPercent)
    }
    // 12/16 Gi MEM = 75%
    if gpuPool.MemPercent != 75 {
        t.Errorf("gpu pool: expected MEM 75%%, got %d%%", gpuPool.MemPercent)
    }
}

func TestAggregateTotal(t *testing.T) {
    pools := []NodePoolInfo{
        {Name: "a", NodeCount: 3, CPUPercent: 50, MemPercent: 60,
            cpuRequests: resource.MustParse("2"), cpuAllocatable: resource.MustParse("4"),
            memRequests: resource.MustParse("6Gi"), memAllocatable: resource.MustParse("10Gi")},
        {Name: "b", NodeCount: 2, CPUPercent: 75, MemPercent: 80,
            cpuRequests: resource.MustParse("6"), cpuAllocatable: resource.MustParse("8"),
            memRequests: resource.MustParse("8Gi"), memAllocatable: resource.MustParse("10Gi")},
    }

    total := ComputeTotal(pools)
    if total.Name != "TOTAL" {
        t.Errorf("expected name 'TOTAL', got %q", total.Name)
    }
    if total.NodeCount != 5 {
        t.Errorf("expected 5 nodes, got %d", total.NodeCount)
    }
    // (2+6)/(4+8) = 8/12 = 66%
    if total.CPUPercent != 66 {
        t.Errorf("expected CPU 66%%, got %d%%", total.CPUPercent)
    }
    // (6+8)/(10+10) = 14/20 Gi = 70%
    if total.MemPercent != 70 {
        t.Errorf("expected MEM 70%%, got %d%%", total.MemPercent)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/collector/`
Expected: compilation error

**Step 3: Write internal/collector/collector.go**

```go
package collector

import (
    "context"
    "sort"
    "sync"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/api/resource"
    "k8s.io/client-go/kubernetes"

    "github.com/ssoriche/kubectl-karpenter/internal/karpenter"
    "github.com/ssoriche/kubectl-karpenter/internal/utilization"
)

// NodePoolInfo contains aggregated utilization for a single NodePool.
type NodePoolInfo struct {
    Name       string
    NodeCount  int
    CPUPercent int
    MemPercent int
    // Kept for total computation
    cpuRequests    resource.Quantity
    cpuAllocatable resource.Quantity
    memRequests    resource.Quantity
    memAllocatable resource.Quantity
}

// Collector fetches cluster data and aggregates per NodePool.
type Collector struct {
    client kubernetes.Interface
}

func NewCollector(client kubernetes.Interface) *Collector {
    return &Collector{client: client}
}

// Collect fetches all nodes and pods, then aggregates by NodePool.
func (c *Collector) Collect(ctx context.Context, selector string) ([]NodePoolInfo, error) {
    var nodes *corev1.NodeList
    var pods *corev1.PodList
    var nodeErr, podErr error

    var wg sync.WaitGroup
    wg.Add(2)
    go func() {
        defer wg.Done()
        opts := metav1.ListOptions{}
        if selector != "" {
            opts.LabelSelector = selector
        }
        nodes, nodeErr = c.client.CoreV1().Nodes().List(ctx, opts)
    }()
    go func() {
        defer wg.Done()
        pods, podErr = c.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
            FieldSelector: "status.phase!=Succeeded,status.phase!=Failed",
        })
    }()
    wg.Wait()

    if nodeErr != nil {
        return nil, nodeErr
    }
    if podErr != nil {
        return nil, podErr
    }

    // Group pods by node
    podsByNode := make(map[string][]corev1.Pod)
    for _, pod := range pods.Items {
        if pod.Spec.NodeName != "" {
            podsByNode[pod.Spec.NodeName] = append(podsByNode[pod.Spec.NodeName], pod)
        }
    }

    return Aggregate(nodes.Items, podsByNode), nil
}

// Aggregate groups nodes by NodePool label and computes utilization.
func Aggregate(nodes []corev1.Node, podsByNode map[string][]corev1.Pod) []NodePoolInfo {
    type poolData struct {
        nodes []corev1.Node
    }
    pools := make(map[string]*poolData)

    for _, node := range nodes {
        poolName, _ := karpenter.GetPoolName(&node)
        if poolName == "" {
            continue // skip non-Karpenter nodes
        }
        if pools[poolName] == nil {
            pools[poolName] = &poolData{}
        }
        pools[poolName].nodes = append(pools[poolName].nodes, node)
    }

    results := make([]NodePoolInfo, 0, len(pools))
    for name, data := range pools {
        info := NodePoolInfo{
            Name:      name,
            NodeCount: len(data.nodes),
        }

        for i := range data.nodes {
            node := &data.nodes[i]
            cpuPct, memPct := utilization.CalculateNodeUtilization(node, podsByNode[node.Name])
            _ = cpuPct
            _ = memPct

            // Accumulate raw values for accurate aggregate percentage
            alloc := node.Status.Allocatable
            if alloc != nil {
                allocCPU := alloc.Cpu().DeepCopy()
                allocMem := alloc.Memory().DeepCopy()
                info.cpuAllocatable.Add(allocCPU)
                info.memAllocatable.Add(allocMem)
            }

            // Sum pod requests for this node
            for _, pod := range podsByNode[node.Name] {
                if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
                    continue
                }
                for _, c := range pod.Spec.Containers {
                    if c.Resources.Requests != nil {
                        if cpu, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
                            info.cpuRequests.Add(cpu)
                        }
                        if mem, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
                            info.memRequests.Add(mem)
                        }
                    }
                }
                for _, c := range pod.Spec.InitContainers {
                    if c.Resources.Requests != nil {
                        if cpu, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
                            info.cpuRequests.Add(cpu)
                        }
                        if mem, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
                            info.memRequests.Add(mem)
                        }
                    }
                }
            }
        }

        // Compute percentages from raw totals
        if !info.cpuAllocatable.IsZero() {
            info.CPUPercent = int((info.cpuRequests.MilliValue() * 100) / info.cpuAllocatable.MilliValue())
        }
        if !info.memAllocatable.IsZero() {
            info.MemPercent = int((info.memRequests.MilliValue() * 100) / info.memAllocatable.MilliValue())
        }

        results = append(results, info)
    }

    sort.Slice(results, func(i, j int) bool {
        return results[i].Name < results[j].Name
    })

    return results
}

// ComputeTotal sums all NodePoolInfo into a total row.
func ComputeTotal(pools []NodePoolInfo) NodePoolInfo {
    total := NodePoolInfo{Name: "TOTAL"}
    for _, p := range pools {
        total.NodeCount += p.NodeCount
        total.cpuRequests.Add(p.cpuRequests)
        total.cpuAllocatable.Add(p.cpuAllocatable)
        total.memRequests.Add(p.memRequests)
        total.memAllocatable.Add(p.memAllocatable)
    }
    if !total.cpuAllocatable.IsZero() {
        total.CPUPercent = int((total.cpuRequests.MilliValue() * 100) / total.cpuAllocatable.MilliValue())
    }
    if !total.memAllocatable.IsZero() {
        total.MemPercent = int((total.memRequests.MilliValue() * 100) / total.memAllocatable.MilliValue())
    }
    return total
}
```

**Step 4: Run tests to verify they pass**

Run: `go test -v ./internal/collector/`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/collector/
git commit -m "feat: add collector with nodepool aggregation"
```

---

### Task 6: Output Printer with Bar Charts

**Files:**
- Create: `internal/output/printer.go`
- Create: `internal/output/printer_test.go`

**Step 1: Write the failing test**

Test table output rendering: given `[]NodePoolInfo`, verify the printed table contains the expected bar output.

```go
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
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/output/`
Expected: compilation error

**Step 3: Write internal/output/printer.go**

```go
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
        fmt.Fprintln(w, "NODEPOOL\tNODES\tCPU\tMEM")
    }

    for _, pool := range pools {
        fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
            pool.Name,
            pool.NodeCount,
            utilization.RenderBar(pool.CPUPercent, barWidth),
            utilization.RenderBar(pool.MemPercent, barWidth),
        )
    }

    // Separator line
    fmt.Fprintln(w, strings.Repeat("─", 70))

    fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
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
```

**Step 4: Run tests to verify they pass**

Run: `go test -v ./internal/output/`
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/output/
git commit -m "feat: add output printer with ASCII bar charts"
```

---

### Task 7: Wire Up main.go with Cobra

**Files:**
- Modify: `cmd/kubectl-karpenter/main.go`

**Step 1: Replace main.go with full Cobra command**

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/spf13/cobra"

    "github.com/ssoriche/kubectl-karpenter/internal/collector"
    "github.com/ssoriche/kubectl-karpenter/internal/kube"
    "github.com/ssoriche/kubectl-karpenter/internal/output"
)

var version = "dev"

func main() {
    if err := newRootCmd().Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

type options struct {
    selector  string
    output    string
    noHeaders bool
}

func newRootCmd() *cobra.Command {
    var opts options

    cmd := &cobra.Command{
        Use:   "kubectl-karpenter [flags]",
        Short: "Show Karpenter NodePool resource utilization",
        Long: `Displays a summary of Karpenter NodePools with node counts and
CPU/memory request utilization shown as ASCII bar charts.

Automatically detects Karpenter API version (v1alpha5, v1beta1, v1).`,
        Example: `  # Show all NodePool utilization
  kubectl karpenter

  # Filter NodePools by label
  kubectl karpenter -l environment=production

  # Output as JSON
  kubectl karpenter -o json`,
        Version:      version,
        SilenceUsage: true,
        RunE: func(cmd *cobra.Command, args []string) error {
            return run(cmd.Context(), opts)
        },
    }

    cmd.Flags().StringVarP(&opts.selector, "selector", "l", "", "Label selector for nodes")
    cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Output format (json, yaml)")
    cmd.Flags().BoolVar(&opts.noHeaders, "no-headers", false, "Don't print headers")

    return cmd
}

func run(ctx context.Context, opts options) error {
    client, err := kube.NewClient()
    if err != nil {
        return fmt.Errorf("failed to create Kubernetes client: %w", err)
    }

    c := collector.NewCollector(client)
    pools, err := c.Collect(ctx, opts.selector)
    if err != nil {
        return fmt.Errorf("failed to collect NodePool data: %w", err)
    }

    if len(pools) == 0 {
        fmt.Fprintln(os.Stderr, "No Karpenter NodePools found")
        return nil
    }

    total := collector.ComputeTotal(pools)
    printer := output.NewPrinter(opts.output, opts.noHeaders)
    return printer.Print(pools, total)
}
```

**Step 2: Verify it compiles**

Run: `make build`
Expected: binary at `bin/kubectl-karpenter`

**Step 3: Run all tests**

Run: `go test -v -race ./...`
Expected: all PASS

**Step 4: Commit**

```bash
git add cmd/kubectl-karpenter/main.go
git commit -m "feat: wire up cobra CLI with full data pipeline"
```

---

### Task 8: Integration Test Against Real Cluster

**Step 1: Manual smoke test**

Run: `bin/kubectl-karpenter`
Expected: table output with NodePool names, node counts, and ASCII bars (or "No Karpenter NodePools found" if no Karpenter cluster)

Run: `bin/kubectl-karpenter -o json`
Expected: JSON output with pools and total

Run: `bin/kubectl-karpenter --no-headers`
Expected: no header row

**Step 2: Verify k9s plugin config works**

Create a test plugin config and verify the command format:
Run: `bin/kubectl-karpenter --help`
Expected: shows usage, flags, examples

**Step 3: Commit any fixes**

If anything needs fixing, fix it and commit.

---

### Task 9: Documentation & k9s Plugin Config

**Files:**
- Create: `deploy/krew/karpenter.yaml`
- Existing: `docs/plans/2026-03-17-kubectl-karpenter-design.md` (already exists)

**Step 1: Create krew manifest**

```yaml
apiVersion: krew.googlecontainertools.github.com/v1alpha2
kind: Plugin
metadata:
  name: karpenter
spec:
  version: v0.1.0
  homepage: https://github.com/ssoriche/kubectl-karpenter
  shortDescription: Show Karpenter NodePool resource utilization
  description: |
    Displays a summary of Karpenter NodePools with node counts and
    CPU/memory request utilization shown as ASCII bar charts.
  platforms:
    - selector:
        matchLabels:
          os: linux
          arch: amd64
      uri: https://github.com/ssoriche/kubectl-karpenter/releases/download/v0.1.0/kubectl-karpenter_linux_amd64.tar.gz
      sha256: "PLACEHOLDER"
      bin: kubectl-karpenter
    - selector:
        matchLabels:
          os: linux
          arch: arm64
      uri: https://github.com/ssoriche/kubectl-karpenter/releases/download/v0.1.0/kubectl-karpenter_linux_arm64.tar.gz
      sha256: "PLACEHOLDER"
      bin: kubectl-karpenter
    - selector:
        matchLabels:
          os: darwin
          arch: amd64
      uri: https://github.com/ssoriche/kubectl-karpenter/releases/download/v0.1.0/kubectl-karpenter_darwin_amd64.tar.gz
      sha256: "PLACEHOLDER"
      bin: kubectl-karpenter
    - selector:
        matchLabels:
          os: darwin
          arch: arm64
      uri: https://github.com/ssoriche/kubectl-karpenter/releases/download/v0.1.0/kubectl-karpenter_darwin_arm64.tar.gz
      sha256: "PLACEHOLDER"
      bin: kubectl-karpenter
    - selector:
        matchLabels:
          os: windows
          arch: amd64
      uri: https://github.com/ssoriche/kubectl-karpenter/releases/download/v0.1.0/kubectl-karpenter_windows_amd64.zip
      sha256: "PLACEHOLDER"
      bin: kubectl-karpenter.exe
```

**Step 2: Commit**

```bash
git add deploy/
git commit -m "feat: add krew plugin manifest"
```

---

### Task 10: Final Verification

**Step 1: Run all checks**

Run: `make check`
Expected: fmt, vet, lint, and tests all pass

**Step 2: Build all platforms**

Run: `make build-all`
Expected: binaries for all platforms in `bin/`

**Step 3: Final commit if needed**

Fix any issues found, commit.
