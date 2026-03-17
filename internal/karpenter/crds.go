package karpenter

import (
	"context"

	"k8s.io/client-go/discovery"
)

// DetectCapabilities checks which Karpenter CRDs exist in the cluster
func DetectCapabilities(ctx context.Context, client discovery.DiscoveryInterface) (*ClusterCapabilities, error) {
	caps := &ClusterCapabilities{}

	_, apiResourceLists, err := client.ServerGroupsAndResources()
	if err != nil {
		if apiResourceLists == nil {
			return nil, err
		}
	}

	for _, list := range apiResourceLists {
		for _, resource := range list.APIResources {
			switch {
			case list.GroupVersion == "karpenter.sh/v1alpha5" && resource.Name == "provisioners":
				caps.HasProvisioners = true
			case list.GroupVersion == "karpenter.sh/v1alpha5" && resource.Name == "machines":
				caps.HasMachines = true
			case (list.GroupVersion == "karpenter.sh/v1beta1" || list.GroupVersion == "karpenter.sh/v1") && resource.Name == "nodepools":
				caps.HasNodePools = true
			case (list.GroupVersion == "karpenter.sh/v1beta1" || list.GroupVersion == "karpenter.sh/v1") && resource.Name == "nodeclaims":
				caps.HasNodeClaims = true
			}
		}
	}

	caps.PrimaryVersion = caps.determinePrimaryVersion()
	return caps, nil
}

func (c *ClusterCapabilities) determinePrimaryVersion() APIVersion {
	if c.HasNodePools || c.HasNodeClaims {
		return APIVersionV1Beta1
	}
	if c.HasProvisioners || c.HasMachines {
		return APIVersionV1Alpha5
	}
	return APIVersionUnknown
}

// HasKarpenter returns true if any Karpenter CRDs are detected
func (c *ClusterCapabilities) HasKarpenter() bool {
	return c.HasNodeClaims || c.HasMachines || c.HasNodePools || c.HasProvisioners
}
