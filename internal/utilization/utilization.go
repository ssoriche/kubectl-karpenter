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
