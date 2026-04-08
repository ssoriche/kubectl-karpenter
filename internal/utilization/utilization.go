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
	for i := range pods {
		if pods[i].Status.Phase == corev1.PodSucceeded || pods[i].Status.Phase == corev1.PodFailed {
			continue
		}
		cpu, mem := EffectivePodRequests(&pods[i])
		totalCPU.Add(cpu)
		totalMem.Add(mem)
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
