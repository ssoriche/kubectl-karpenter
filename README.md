# kubectl-karpenter

A kubectl plugin that shows Karpenter NodePool resource utilization.

## Features

- Shows node counts and CPU/memory request utilization per NodePool
- ASCII bar charts for quick visual assessment
- Automatically detects Karpenter API version (v1alpha5, v1beta1, v1)
- Supports mixed-version clusters during migrations
- Outputs in table, JSON, or YAML format

## Installation

### Via krew (recommended)

```bash
kubectl krew install karpenter
```

### Manual installation

Download the appropriate binary from the [releases page](https://github.com/ssoriche/kubectl-karpenter/releases), extract it, and place it in your PATH.

```bash
# Example for macOS (Apple Silicon)
tar -xzf kubectl-karpenter_darwin_arm64.tar.gz
chmod +x kubectl-karpenter
sudo mv kubectl-karpenter /usr/local/bin/
```

## Usage

```bash
# Show all NodePool utilization
kubectl karpenter

# Filter nodes by label
kubectl karpenter -l environment=production

# Output as JSON
kubectl karpenter -o json

# Output as YAML
kubectl karpenter -o yaml

# Hide column headers
kubectl karpenter --no-headers
```

## Output Example

```
NODEPOOL    NODES  CPU                      MEM
default       3    [████░░░░░░░░░░] 40%    [███████░░░░░░░] 70%
compute       2    [████████░░░░░░] 65%    [█████░░░░░░░░░] 45%
──────────────────────────────────────────────────────────────────────
TOTAL         5    [█████░░░░░░░░░] 50%    [██████░░░░░░░░] 60%
```

## Karpenter Version Support

The plugin automatically detects which Karpenter version is installed:

| Version | Node Labels | Column Header |
|---------|-------------|---------------|
| v1alpha5 | `karpenter.sh/provisioner-name` | PROVISIONER |
| v1beta1/v1 | `karpenter.sh/nodepool` | NODEPOOL |

Mixed-version clusters are supported during migrations.

## Development

```bash
# Enter devbox shell
devbox shell

# Build
make build

# Run tests
make test

# Run linter
make lint

# Build for all platforms
make build-all
```

## License

MIT
