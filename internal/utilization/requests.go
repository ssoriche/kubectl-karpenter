package utilization

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// EffectivePodRequests computes the effective resource requests for a pod
// according to the Kubernetes scheduling algorithm (KEP-753).
//
// The effective request accounts for:
//   - Regular init containers that run sequentially and exit
//   - Sidecar init containers (restartPolicy=Always) that persist
//   - Regular containers that all run concurrently
//
// For each resource, the effective request is:
//
//	max(peak_during_init_phase, sum_regular_containers + sum_sidecar_containers)
func EffectivePodRequests(pod *corev1.Pod) (cpu, mem resource.Quantity) {
	var maxInitCPU, maxInitMem int64
	var sidecarCPU, sidecarMem int64

	for _, ic := range pod.Spec.InitContainers {
		icCPU := requestMilliValue(ic, corev1.ResourceCPU)
		icMem := requestMilliValue(ic, corev1.ResourceMemory)

		if ic.RestartPolicy != nil && *ic.RestartPolicy == corev1.ContainerRestartPolicyAlways {
			// Sidecar: starts and keeps running alongside subsequent containers
			sidecarCPU += icCPU
			sidecarMem += icMem
			if sidecarCPU > maxInitCPU {
				maxInitCPU = sidecarCPU
			}
			if sidecarMem > maxInitMem {
				maxInitMem = sidecarMem
			}
		} else {
			// Regular init: runs then exits; concurrent with already-started sidecars
			if c := sidecarCPU + icCPU; c > maxInitCPU {
				maxInitCPU = c
			}
			if c := sidecarMem + icMem; c > maxInitMem {
				maxInitMem = c
			}
		}
	}

	var regularCPU, regularMem int64
	for _, c := range pod.Spec.Containers {
		regularCPU += requestMilliValue(c, corev1.ResourceCPU)
		regularMem += requestMilliValue(c, corev1.ResourceMemory)
	}

	effectiveCPU := maxInitCPU
	if v := sidecarCPU + regularCPU; v > effectiveCPU {
		effectiveCPU = v
	}
	effectiveMem := maxInitMem
	if v := sidecarMem + regularMem; v > effectiveMem {
		effectiveMem = v
	}

	cpu = *resource.NewMilliQuantity(effectiveCPU, resource.DecimalSI)
	mem = *resource.NewMilliQuantity(effectiveMem, resource.DecimalSI)
	return cpu, mem
}

func requestMilliValue(c corev1.Container, name corev1.ResourceName) int64 {
	if c.Resources.Requests == nil {
		return 0
	}
	if v, ok := c.Resources.Requests[name]; ok {
		return v.MilliValue()
	}
	return 0
}
