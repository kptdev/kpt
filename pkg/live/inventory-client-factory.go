package live

import (
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/inventory"
)

// ClusterClientFactory is a factory that creates instances of ClusterClient inventory client.
type ClusterClientFactory struct {
	StatusPolicy inventory.StatusPolicy
}

func NewClusterClientFactory() *ClusterClientFactory {
	return &ClusterClientFactory{StatusPolicy: inventory.StatusPolicyNone}
}
func (ccf *ClusterClientFactory) NewClient(factory cmdutil.Factory) (inventory.Client, error) {
	return inventory.NewClient(factory, WrapInventoryObj, InvToUnstructuredFunc, ccf.StatusPolicy, ResourceGroupGVK)
}
