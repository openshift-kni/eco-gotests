package nad

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/msg"
	"k8s.io/utils/strings/slices"
)

var (
	// allowedMacVlanMode represents all allowed modes for macvlan plugin type.
	allowedMacVlanMode = []string{"bridge", "passthru", "private", "vepa"}
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

		plugin.errorMsg = "invalid ipam parameter"
	}

	plugin.masterPlugin.Ipam = ipam

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

		plugin.errorMsg = "invalid ipam parameter"
	}

	plugin.masterPlugin.Ipam = ipam

	return plugin
}
