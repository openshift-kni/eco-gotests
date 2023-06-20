package define

import (
	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/nad"
)

// MasterNadPlugin sets NetworkAttachmentDefinition master plugin based on given input.
func MasterNadPlugin(nadName, mode string, ipam *nad.IPAM, masterInterface ...string) (*nad.MasterPlugin, error) {
	glog.V(90).Infof("Defining master nad plugin with the following parameters name %s, mode %s, ipam %v",
		nadName, mode, ipam)

	macVlan := nad.NewMasterMacVlanPlugin(nadName)

	if len(masterInterface) > 0 {
		glog.V(90).Infof("Attaching master nad plugin to interface %s", masterInterface[0])
		macVlan.WithMasterInterface(masterInterface[0])
	}

	macVlan.WithMode(mode).WithIPAM(ipam)

	return macVlan.GetMasterPluginConfig()
}
