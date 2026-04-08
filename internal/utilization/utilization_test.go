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
			name: "regular init container larger than regular",
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
						InitContainers: []corev1.Container{{
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("4Gi"),
								},
							},
						}},
						Containers: []corev1.Container{{
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("2Gi"),
								},
							},
						}},
					},
				},
			},
			// Effective = max(init=1, regular=500m) = 1 CPU -> 25%
			expectedCPU: 25,
			// Effective = max(init=4Gi, regular=2Gi) = 4Gi -> 50%
			expectedMem: 50,
		},
		{
			name: "sidecar init container adds to regular",
			node: &corev1.Node{
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("8Gi"),
					},
				},
			},
			pods: func() []corev1.Pod {
				restartAlways := corev1.ContainerRestartPolicyAlways
				return []corev1.Pod{
					{
						Status: corev1.PodStatus{Phase: corev1.PodRunning},
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{{
								RestartPolicy: &restartAlways,
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
										corev1.ResourceCPU:    resource.MustParse("1"),
										corev1.ResourceMemory: resource.MustParse("2Gi"),
									},
								},
							}},
						},
					},
				}
			}(),
			// Effective = sidecar(1) + regular(1) = 2 CPU -> 50%
			expectedCPU: 50,
			// Effective = sidecar(2Gi) + regular(2Gi) = 4Gi -> 50%
			expectedMem: 50,
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
