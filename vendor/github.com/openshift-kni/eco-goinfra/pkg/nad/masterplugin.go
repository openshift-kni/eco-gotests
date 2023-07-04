package nad

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	"k8s.io/utils/strings/slices"
)

var (
	// allowedMacVlanMode represents all allowed modes for macvlan plugin type.
	allowedMacVlanMode      = []string{"bridge", "passthru", "private", "vepa"}
	invalidIpamParameterMsg = "invalid ipam parameter"
)

// MasterMacVlanPlugin provides struct for NetworkAttachmentDefinition Master plugin with macvlan configuration.
type MasterMacVlanPlugin struct {
	masterPlugin *MasterPlugin
	errorMsg     string
}

// NewMasterMacVlanPlugin creates new instance of MasterMacVlanPlugin.
func NewMasterMacVlanPlugin(name string) *MasterMacVlanPlugin {
	glog.V(100).Infof(
		"Initializing new MasterVlanPlugin structure with the following param: %s", name)

	builder := MasterMacVlanPlugin{
		masterPlugin: &MasterPlugin{
			CniVersion: "0.3.1",
			Name:       name,
			Type:       "macvlan",
		},
	}

	if builder.masterPlugin.Name == "" {
		glog.V(100).Infof("error MasterMacVlanPlugin can not be empty")

		builder.errorMsg = "MasterMacVlanPlugin name is empty"
	}

	return &builder
}

// WithMode defines macvlan type to MasterMacVlanPlugin. Default is bridge.
func (plugin *MasterMacVlanPlugin) WithMode(mode string) *MasterMacVlanPlugin {
	glog.V(100).Infof("Adding macvlan mode %s to MasterMacVlanPlugin", mode)

	if !slices.Contains(allowedMacVlanMode, mode) {
		glog.V(100).Infof("error to add mode %s, allowed modes are %v", mode, allowedMacVlanMode)

		plugin.errorMsg = "invalid mode parameter"
	}

	if plugin.masterPlugin == nil {
		glog.V(100).Infof(msg.UndefinedCrdObjectErrString("MasterMacVlanPlugin"))
		plugin.errorMsg = msg.UndefinedCrdObjectErrString("MasterMacVlanPlugin")
	}

	plugin.masterPlugin.Mode = mode

	return plugin
}

// WithMasterInterface defines master interface to MasterMacVlanPlugin. Default is cn0.
func (plugin *MasterMacVlanPlugin) WithMasterInterface(master string) *MasterMacVlanPlugin {
	glog.V(100).Infof("Adding master interface %s to MasterMacVlanPlugin", master)

	if plugin.masterPlugin == nil {
		glog.V(100).Infof(msg.UndefinedCrdObjectErrString("MasterMacVlanPlugin"))
		plugin.errorMsg = msg.UndefinedCrdObjectErrString("MasterMacVlanPlugin")
	}

	if master == "" {
		glog.V(100).Infof("error to add master interface, the name of interface can not be empty")

		plugin.errorMsg = "invalid master parameter"
	}

	plugin.masterPlugin.Master = master

	return plugin
}

// WithIPAM defines IPAM configuration to MasterMacVlanPlugin. Default is empty.
func (plugin *MasterMacVlanPlugin) WithIPAM(ipam *IPAM) *MasterMacVlanPlugin {
	glog.V(100).Infof("Adding ipam configuration %v to MasterMacVlanPlugin", ipam)

	if plugin.masterPlugin == nil {
		glog.V(100).Infof(msg.UndefinedCrdObjectErrString("MasterMacVlanPlugin"))
		plugin.errorMsg = msg.UndefinedCrdObjectErrString("MasterMacVlanPlugin")
	}

	if ipam == nil {
		glog.V(100).Infof("error to add empty ipam to MasterMacVlanPlugin")

		plugin.errorMsg = invalidIpamParameterMsg
	}

	plugin.masterPlugin.Ipam = ipam

	return plugin
}

// WithLinkInContainer defines MasterMacVlan plugin using linkInContainer feature.
func (plugin *MasterMacVlanPlugin) WithLinkInContainer() *MasterMacVlanPlugin {
	glog.V(100).Infof("Adding linkInContainer configuration to MasterMacVlanPlugin")

	if plugin.masterPlugin == nil {
		glog.V(100).Infof(msg.UndefinedCrdObjectErrString("MasterMacVlanPlugin"))
		plugin.errorMsg = msg.UndefinedCrdObjectErrString("MasterMacVlanPlugin")
	}

	plugin.masterPlugin.LinkInContainer = true

	return plugin
}

// GetMasterPluginConfig returns master plugin if error is not occur.
func (plugin *MasterMacVlanPlugin) GetMasterPluginConfig() (*MasterPlugin, error) {
	if plugin.errorMsg != "" {
		return nil, fmt.Errorf("error to build MaterPlugin config due to :%s", plugin.errorMsg)
	}

	return plugin.masterPlugin, nil
}

// MasterBridgePlugin provides struct for MasterPlugin set to bridge in NetworkAttachmentDefinition.
type MasterBridgePlugin struct {
	masterPlugin *MasterPlugin
	errorMsg     string
}

// NewMasterBridgePlugin creates new instance of MasterBridgePlugin.
func NewMasterBridgePlugin(name, bridgeName string) *MasterBridgePlugin {
	glog.V(100).Infof(
		"Initializing new MasterBridgePlugin structure %s, with bridge %s", name, bridgeName)

	builder := MasterBridgePlugin{
		masterPlugin: &MasterPlugin{
			CniVersion: "0.3.1",
			Name:       name,
			Type:       "bridge",
			Bridge:     bridgeName,
		},
	}

	if builder.masterPlugin.Name == "" {
		glog.V(100).Infof("error MasterBridgePlugin can not be empty")

		builder.errorMsg = "MasterBridgePlugin name is empty"
	}

	return &builder
}

// GetMasterPluginConfig returns master plugin if error does not occur.
func (plugin *MasterBridgePlugin) GetMasterPluginConfig() (*MasterPlugin, error) {
	if plugin.errorMsg != "" {
		return nil, fmt.Errorf("error to build MaterPlugin config due to :%s", plugin.errorMsg)
	}

	return plugin.masterPlugin, nil
}

// WithIPAM defines IPAM configuration to MasterBridgePlugin. Default is empty.
func (plugin *MasterBridgePlugin) WithIPAM(ipam *IPAM) *MasterBridgePlugin {
	glog.V(100).Infof("Adding ipam configuration %v to MasterBridgePlugin", ipam)

	if plugin.masterPlugin == nil {
		glog.V(100).Infof(msg.UndefinedCrdObjectErrString("MasterBridgePlugin"))
		plugin.errorMsg = msg.UndefinedCrdObjectErrString("MasterBridgePlugin")
	}

	if ipam == nil {
		glog.V(100).Infof("error adding empty ipam to MasterBridgePlugin")

		plugin.errorMsg = invalidIpamParameterMsg
	}

	plugin.masterPlugin.Ipam = ipam

	return plugin
}

// MasterVlanPlugin provides struct for MasterPlugin set to vlan in NetworkAttachmentDefinition.
type MasterVlanPlugin struct {
	masterPlugin *MasterPlugin
	errorMsg     string
}

// NewMasterVlanPlugin creates new instance of MasterVlanPlugin.
func NewMasterVlanPlugin(name string, vlanID uint16) *MasterVlanPlugin {
	glog.V(100).Infof(
		"Initializing new MasterVlanPlugin structure %s, with vlanId %s", vlanID)

	builder := MasterVlanPlugin{
		masterPlugin: &MasterPlugin{
			CniVersion: "0.3.1",
			Name:       name,
			Type:       "vlan",
			VlanID:     vlanID,
		},
	}

	if vlanID > 4094 {
		glog.V(100).Infof("error vlan id can not be greater than 4094")

		builder.errorMsg = "MasterVlanPlugin vlanID is greater than 4094"
	}

	if builder.masterPlugin.Name == "" {
		glog.V(100).Infof("error MasterVlanPlugin name can not be empty")

		builder.errorMsg = "MasterVlanPlugin name is empty"
	}

	return &builder
}

// WithIPAM defines IPAM configuration to MasterVlanPlugin. Default is empty.
func (plugin *MasterVlanPlugin) WithIPAM(ipam *IPAM) *MasterVlanPlugin {
	glog.V(100).Infof("Adding IPAM configuration to MasterVlanPlugin: %v", ipam)

	if plugin.masterPlugin == nil {
		glog.V(100).Infof(msg.UndefinedCrdObjectErrString("MasterVlanPlugin"))
		plugin.errorMsg = msg.UndefinedCrdObjectErrString("MasterVlanPlugin")
	}

	if ipam == nil {
		glog.V(100).Infof("error adding empty ipam to MasterVlanPlugin")

		plugin.errorMsg = invalidIpamParameterMsg
	}

	if plugin.errorMsg != "" {
		return plugin
	}

	plugin.masterPlugin.Ipam = ipam

	return plugin
}

// WithMasterInterface defines master interface to MasterVlanPlugin. Default is cn0.
func (plugin *MasterVlanPlugin) WithMasterInterface(masterInterfaceName string) *MasterVlanPlugin {
	glog.V(100).Infof("Adding masterInterfaceName interface %s to MasterVlanPlugin", masterInterfaceName)

	if plugin.masterPlugin == nil {
		glog.V(100).Infof(msg.UndefinedCrdObjectErrString("MasterVlanPlugin"))
		plugin.errorMsg = msg.UndefinedCrdObjectErrString("MasterVlanPlugin")
	}

	if masterInterfaceName == "" {
		glog.V(100).Infof("error to add masterInterfaceName interface, the name of interface can not be empty")

		plugin.errorMsg = "invalid masterInterfaceName parameter"
	}

	if plugin.errorMsg != "" {
		return plugin
	}

	plugin.masterPlugin.Master = masterInterfaceName

	return plugin
}

// WithLinkInContainer defines MasterVlanPlugin using linkInContainer feature.
func (plugin *MasterVlanPlugin) WithLinkInContainer() *MasterVlanPlugin {
	if plugin.masterPlugin == nil {
		glog.V(100).Infof(msg.UndefinedCrdObjectErrString("MasterVlanPlugin"))
		plugin.errorMsg = msg.UndefinedCrdObjectErrString("MasterVlanPlugin")
	}

	if plugin.errorMsg != "" {
		return plugin
	}

	plugin.masterPlugin.LinkInContainer = true

	return plugin
}

// GetMasterPluginConfig returns master plugin if error does not occur.
func (plugin *MasterVlanPlugin) GetMasterPluginConfig() (*MasterPlugin, error) {
	if plugin.errorMsg != "" {
		return nil, fmt.Errorf("error to build MaterPlugin config due to :%s", plugin.errorMsg)
	}

	return plugin.masterPlugin, nil
}
