package karpenter

// Karpenter v1alpha5 labels
const (
	LabelProvisionerName = "karpenter.sh/provisioner-name"
)

// Karpenter v1beta1/v1 labels
const (
	LabelNodePool = "karpenter.sh/nodepool"
)

// Common labels (all versions)
const (
	LabelCapacityType = "karpenter.sh/capacity-type"
	LabelInstanceType = "node.kubernetes.io/instance-type"
	LabelTopologyZone = "topology.kubernetes.io/zone"
)

// Capacity types
const (
	CapacityTypeOnDemand = "on-demand"
	CapacityTypeSpot     = "spot"
)

// Annotations (all versions)
const (
	AnnotationDoNotEvict       = "karpenter.sh/do-not-evict"
	AnnotationDoNotDisrupt     = "karpenter.sh/do-not-disrupt"
	AnnotationDoNotConsolidate = "karpenter.sh/do-not-consolidate"
)

// CRD names
const (
	CRDNodeClaims   = "nodeclaims.karpenter.sh"
	CRDMachines     = "machines.karpenter.sh"
	CRDNodePools    = "nodepools.karpenter.sh"
	CRDProvisioners = "provisioners.karpenter.sh"
)
