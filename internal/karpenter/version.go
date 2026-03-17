package karpenter

import (
	corev1 "k8s.io/api/core/v1"
)

// APIVersion represents a Karpenter API version
type APIVersion string

const (
	APIVersionV1Alpha5 APIVersion = "v1alpha5"
	APIVersionV1Beta1  APIVersion = "v1beta1"
	APIVersionV1       APIVersion = "v1"
	APIVersionUnknown  APIVersion = "unknown"
)

// ClusterCapabilities represents which Karpenter CRDs are available in the cluster
type ClusterCapabilities struct {
	HasNodeClaims   bool       // v1beta1/v1
	HasMachines     bool       // v1alpha5
	HasNodePools    bool       // v1beta1/v1
	HasProvisioners bool       // v1alpha5
	PrimaryVersion  APIVersion // Most likely version based on CRDs
}

// DetectNodeVersion determines which API version provisioned a specific node
// based on its labels
func DetectNodeVersion(node *corev1.Node) APIVersion {
	if node == nil || node.Labels == nil {
		return APIVersionUnknown
	}

	// v1beta1/v1 uses karpenter.sh/nodepool
	if _, ok := node.Labels[LabelNodePool]; ok {
		return APIVersionV1Beta1
	}

	// v1alpha5 uses karpenter.sh/provisioner-name
	if _, ok := node.Labels[LabelProvisionerName]; ok {
		return APIVersionV1Alpha5
	}

	return APIVersionUnknown
}

// GetPoolName returns the nodepool or provisioner name from node labels
// along with the detected API version
func GetPoolName(node *corev1.Node) (name string, version APIVersion) {
	if node == nil || node.Labels == nil {
		return "", APIVersionUnknown
	}

	// Check v1beta1/v1 first (newer)
	if name, ok := node.Labels[LabelNodePool]; ok {
		return name, APIVersionV1Beta1
	}

	// Fall back to v1alpha5
	if name, ok := node.Labels[LabelProvisionerName]; ok {
		return name, APIVersionV1Alpha5
	}

	return "", APIVersionUnknown
}

// DeterminePoolColumnHeader returns the appropriate column header based on cluster capabilities
func (c *ClusterCapabilities) DeterminePoolColumnHeader() string {
	if c.HasNodePools || c.HasNodeClaims {
		return "NODEPOOL"
	}
	if c.HasProvisioners || c.HasMachines {
		return "PROVISIONER"
	}
	return "NODEPOOL"
}
