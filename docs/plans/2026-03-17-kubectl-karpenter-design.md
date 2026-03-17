# kubectl-karpenter Design

## Overview

A kubectl plugin that displays Karpenter NodePool resource utilization at a glance.
Static one-shot table output with ASCII utilization bars, pipe-friendly and compatible
with k9s plugin integration.

## CLI Interface

```
$ kubectl karpenter
NODEPOOL       NODES   CPU                     MEM
default        12      [████████░░░░░░] 55%    [██████████░░░░] 71%
gpu-workers     4      [███████████░░░] 82%    [████████████░░] 88%
spot-batch      8      [███░░░░░░░░░░░] 21%    [█████░░░░░░░░░] 35%
─────────────────────────────────────────────────────────────────────
TOTAL          24      [██████░░░░░░░░] 48%    [████████░░░░░░] 61%
```

### Flags

- `--output / -o` — `table` (default), `json`, `yaml`
- `--no-headers` — suppress header row
- `--selector / -l` — filter NodePools by label selector

Single root command, no subcommands.

## Architecture

```
kubectl-karpenter/
├── cmd/kubectl-karpenter/
│   └── main.go              # Cobra root command
├── internal/
│   ├── kube/
│   │   └── client.go        # Kubeconfig -> clientset
│   ├── karpenter/
│   │   ├── nodepools.go     # List NodePools, detect API version
│   │   └── version.go       # v1alpha5/v1beta1/v1 detection
│   ├── collector/
│   │   └── collector.go     # Orchestrate: fetch nodes, pods, aggregate
│   ├── utilization/
│   │   └── utilization.go   # Sum pod requests vs node allocatable
│   └── output/
│       └── printer.go       # Table (with bars), JSON, YAML
├── deploy/krew/
├── devbox.json
├── Makefile
└── .goreleaser.yaml
```

## Data Flow

1. Detect Karpenter API version via discovery (v1alpha5 -> Provisioners, v1beta1/v1 -> NodePools)
2. List all NodePools (or Provisioners)
3. List all Nodes, group by NodePool label (`karpenter.sh/nodepool` or `karpenter.sh/provisioner-name`)
4. List all Pods, group by node
5. For each NodePool: sum pod CPU/MEM requests, sum node allocatable CPU/MEM, compute percentages
6. Render table with bars, append total row

All API calls are batch (list all nodes in one call, all pods in one call).

## Utilization Calculation

- Per NodePool: sum all pod `resources.requests.cpu` and `resources.requests.memory` across all nodes in the pool
- Divide by sum of `status.allocatable.cpu` and `status.allocatable.memory` for those nodes
- Total row: sum across all NodePools

### Bar Rendering

14-character bar width (~7% per block):

```
[██████████████] 100%
[██████████░░░░]  71%
[░░░░░░░░░░░░░░]   0%
```

Filled = `█`, empty = `░`. Percentage right-aligned after the bar.

### Edge Cases

- NodePool with 0 nodes: show `0` nodes, no bar, `0%`
- Pods in `Succeeded`/`Failed` phase: excluded (not consuming resources)
- DaemonSet pods: included (they consume allocatable capacity)

## Karpenter Version Support

Multi-version support via discovery API:

- **v1alpha5**: Provisioners, `karpenter.sh/provisioner-name` node label
- **v1beta1/v1**: NodePools, `karpenter.sh/nodepool` node label

## k9s Integration

Users add to `~/.config/k9s/plugins.yml`:

```yaml
plugins:
  karpenter:
    shortCut: Shift-K
    description: Show Karpenter NodePool utilization
    scopes:
      - all
    command: kubectl
    args:
      - karpenter
    background: false
```

## Technology

- Go (matching kubectl-consolidation patterns)
- Cobra for CLI
- text/tabwriter for table output
- client-go for Kubernetes API
- Requests-based utilization (no metrics-server dependency)
- Krew for distribution
