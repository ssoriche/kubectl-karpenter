package collector

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestAggregateWithInitContainers(t *testing.T) {
	restartAlways := corev1.ContainerRestartPolicyAlways

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
	}

	podsByNode := map[string][]corev1.Pod{
		"node-1": {
			// Pod with regular init container (1 CPU) larger than regular container (500m)
			// Effective CPU = max(1, 0.5) = 1 CPU = 25%
			{
				Status: corev1.PodStatus{Phase: corev1.PodRunning},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1"),
								corev1.ResourceMemory: resource.MustParse("2Gi"),
							},
						},
					}},
					Containers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					}},
				},
			},
			// Pod with sidecar init container (500m) + regular container (500m)
			// Effective CPU = sidecar(500m) + regular(500m) = 1 CPU = 25%
			{
				Status: corev1.PodStatus{Phase: corev1.PodRunning},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{
						RestartPolicy: &restartAlways,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					}},
					Containers: []corev1.Container{{
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					}},
				},
			},
		},
	}

	results := Aggregate(nodes, podsByNode)

	if len(results) != 1 {
		t.Fatalf("expected 1 pool, got %d", len(results))
	}

	pool := results[0]
	// Total effective: 1 CPU (init pod) + 1 CPU (sidecar pod) = 2 CPU out of 4 = 50%
	if pool.CPUPercent != 50 {
		t.Errorf("expected CPU 50%%, got %d%%", pool.CPUPercent)
	}
	// Total effective: 2Gi (init pod) + 2Gi (sidecar pod) = 4Gi out of 8Gi = 50%
	if pool.MemPercent != 50 {
		t.Errorf("expected MEM 50%%, got %d%%", pool.MemPercent)
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
