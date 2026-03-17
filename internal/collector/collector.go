package collector

import (
	"context"
	"sort"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ssoriche/kubectl-karpenter/internal/karpenter"
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
