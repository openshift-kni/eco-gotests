package define

import (
	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
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

// TapNad defines and creates tap NetworkAttachmentDefinition on a cluster.
func TapNad(
	apiClient *clients.Settings,
	name string,
	nsname string,
	user int,
	group int,
	sysctlConfig map[string]string) (*nad.Builder, error) {
	plugins := []nad.Plugin{*nad.TapPlugin(user, group, true)}

	if sysctlConfig != nil {
		plugins = append(plugins, *nad.TuningSysctlPlugin(true, sysctlConfig))
	}

	tap, err := nad.NewBuilder(apiClient, name, nsname).WithPlugins(name, &plugins).Create()

	if err != nil {
		return nil, err
	}

	return tap, nil
}

// MacVlanNad defines and creates mac-vlan NetworkAttachmentDefinition on a cluster.
func MacVlanNad(apiClient *clients.Settings, name, nsName, intName string, ipam *nad.IPAM) (*nad.Builder, error) {
	masterPlugin, err := nad.NewMasterMacVlanPlugin(name).WithMasterInterface(intName).
		WithIPAM(ipam).WithLinkInContainer().GetMasterPluginConfig()
	if err != nil {
		return nil, err
	}

	return createNadWithMasterPlugin(apiClient, name, nsName, masterPlugin)
}

// VlanNad defines and creates Vlan NetworkAttachmentDefinition on a cluster.
func VlanNad(
	apiClient *clients.Settings, name, nsName, intName string, vlanID uint16, ipam *nad.IPAM) (*nad.Builder, error) {
	masterPlugin, err := nad.NewMasterVlanPlugin(name, vlanID).WithMasterInterface(intName).
		WithIPAM(ipam).WithLinkInContainer().GetMasterPluginConfig()

	if err != nil {
		return nil, err
	}

	return createNadWithMasterPlugin(apiClient, name, nsName, masterPlugin)
}

// IPVlanNad defines and creates IP-Vlan NetworkAttachmentDefinition on a cluster.
func IPVlanNad(apiClient *clients.Settings, name, nsName, intName string, ipam *nad.IPAM) (*nad.Builder, error) {
	masterPlugin, err := nad.NewMasterIPVlanPlugin(name).WithMasterInterface(intName).WithIPAM(ipam).
		WithLinkInContainer().GetMasterPluginConfig()
	if err != nil {
		return nil, err
	}

	return createNadWithMasterPlugin(apiClient, name, nsName, masterPlugin)
}

func createNadWithMasterPlugin(
	apiClient *clients.Settings, name, nsName string, masterPlugin *nad.MasterPlugin) (*nad.Builder, error) {
	createdNad, err := nad.NewBuilder(apiClient, name, nsName).WithMasterPlugin(masterPlugin).Create()
	if err != nil {
		return nil, err
	}

	return createdNad, nil
}
