package utilization

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func restartPolicyAlways() *corev1.ContainerRestartPolicy {
	p := corev1.ContainerRestartPolicyAlways
	return &p
}

func container(cpu, mem string) corev1.Container {
	return corev1.Container{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(mem),
			},
		},
	}
}

func initContainer(cpu, mem string) corev1.Container {
	return container(cpu, mem)
}

func sidecarContainer(cpu, mem string) corev1.Container {
	c := container(cpu, mem)
	c.RestartPolicy = restartPolicyAlways()
	return c
}

func TestEffectivePodRequests(t *testing.T) {
	tests := []struct {
		name    string
		pod     *corev1.Pod
		wantCPU string
		wantMem string
	}{
		{
			name:    "empty pod",
			pod:     &corev1.Pod{},
			wantCPU: "0",
			wantMem: "0",
		},
		{
			name: "regular containers only",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						container("200m", "128Mi"),
						container("300m", "256Mi"),
					},
				},
			},
			wantCPU: "500m",
			wantMem: "384Mi",
		},
		{
			name: "regular init larger than regular containers",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						initContainer("1", "1Gi"),
					},
					Containers: []corev1.Container{
						container("200m", "128Mi"),
					},
				},
			},
			wantCPU: "1",
			wantMem: "1Gi",
		},
		{
			name: "regular init smaller than regular containers",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						initContainer("100m", "64Mi"),
					},
					Containers: []corev1.Container{
						container("200m", "128Mi"),
						container("300m", "256Mi"),
					},
				},
			},
			wantCPU: "500m",
			wantMem: "384Mi",
		},
		{
			name: "multiple regular init containers takes max",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						initContainer("500m", "512Mi"),
						initContainer("800m", "1Gi"),
					},
					Containers: []corev1.Container{
						container("200m", "128Mi"),
					},
				},
			},
			wantCPU: "800m",
			wantMem: "1Gi",
		},
		{
			name: "single sidecar persists into regular phase",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						sidecarContainer("300m", "256Mi"),
					},
					Containers: []corev1.Container{
						container("200m", "128Mi"),
					},
				},
			},
			wantCPU: "500m",
			wantMem: "384Mi",
		},
		{
			name: "sidecar before regular init",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						sidecarContainer("200m", "128Mi"),
						initContainer("500m", "512Mi"),
					},
					Containers: []corev1.Container{
						container("100m", "64Mi"),
					},
				},
			},
			// Init phase peak: sidecar(200m) + init(500m) = 700m
			// Regular phase: sidecar(200m) + regular(100m) = 300m
			// Effective = max(700m, 300m) = 700m
			wantCPU: "700m",
			// Init phase peak: 128Mi + 512Mi = 640Mi
			// Regular phase: 128Mi + 64Mi = 192Mi
			// Effective = max(640Mi, 192Mi) = 640Mi
			wantMem: "640Mi",
		},
		{
			name: "regular init before sidecar",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						initContainer("500m", "512Mi"),
						sidecarContainer("200m", "128Mi"),
					},
					Containers: []corev1.Container{
						container("100m", "64Mi"),
					},
				},
			},
			// Init phase: max(500m, sidecar(200m)) = 500m (regular init runs alone, no sidecars yet)
			// Regular phase: sidecar(200m) + regular(100m) = 300m
			// Effective = max(500m, 300m) = 500m
			wantCPU: "500m",
			wantMem: "512Mi",
		},
		{
			name: "multiple sidecars all persist",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						sidecarContainer("100m", "64Mi"),
						sidecarContainer("200m", "128Mi"),
					},
					Containers: []corev1.Container{
						container("300m", "256Mi"),
					},
				},
			},
			// All sidecars persist: 100m + 200m + 300m = 600m
			wantCPU: "600m",
			wantMem: "448Mi",
		},
		{
			name: "mixed sidecar regular sidecar",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						sidecarContainer("100m", "64Mi"),
						initContainer("400m", "256Mi"),
						sidecarContainer("150m", "128Mi"),
					},
					Containers: []corev1.Container{
						container("200m", "64Mi"),
					},
				},
			},
			// Step 1: sidecar 100m -> sidecarSum=100m, maxInit=100m
			// Step 2: regular init 400m -> candidate=100m+400m=500m, maxInit=500m
			// Step 3: sidecar 150m -> sidecarSum=250m, maxInit=max(500m,250m)=500m
			// Regular phase: sidecarSum(250m) + regular(200m) = 450m
			// Effective = max(500m, 450m) = 500m
			wantCPU: "500m",
			// Step 1: sidecar 64Mi -> sidecarSum=64Mi, maxInit=64Mi
			// Step 2: regular init 256Mi -> candidate=64Mi+256Mi=320Mi, maxInit=320Mi
			// Step 3: sidecar 128Mi -> sidecarSum=192Mi, maxInit=max(320Mi,192Mi)=320Mi
			// Regular phase: 192Mi + 64Mi = 256Mi
			// Effective = max(320Mi, 256Mi) = 320Mi
			wantMem: "320Mi",
		},
		{
			name: "nil requests on containers",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{},
					},
					Containers: []corev1.Container{
						{},
					},
				},
			},
			wantCPU: "0",
			wantMem: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCPU, gotMem := EffectivePodRequests(tt.pod)

			wantCPU := resource.MustParse(tt.wantCPU)
			wantMem := resource.MustParse(tt.wantMem)

			if gotCPU.Cmp(wantCPU) != 0 {
				t.Errorf("CPU = %s, want %s", gotCPU.String(), wantCPU.String())
			}
			if gotMem.Cmp(wantMem) != 0 {
				t.Errorf("Mem = %s, want %s", gotMem.String(), wantMem.String())
			}
		})
	}
}
