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
