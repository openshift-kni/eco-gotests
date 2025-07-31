package deviceconfig

import (
	"fmt"

	"github.com/openshift-kni/eco-goinfra/pkg/amdgpu"
)

// GetEnableNodeLabeller - Get the value of 'enableNodeLabeller' from the deviceConfig.
func GetEnableNodeLabeller(builder *amdgpu.Builder) *bool {
	return builder.Object.Spec.DevicePlugin.EnableNodeLabeller
}

// IsNodeLabellerEnabled - The function returns a boolean value indicating the
// 'enableNodeLabeller' key under 'spec.devicePlugin' in the DeviceConfig.
// Note: If the value of that key is 'nil', a boolean value of 'false' will be returned.
func IsNodeLabellerEnabled(builder *amdgpu.Builder) bool {
	enableNodeLabeller := GetEnableNodeLabeller(builder)

	isNodeLabellerEnabled := enableNodeLabeller != nil && *enableNodeLabeller

	return isNodeLabellerEnabled
}

// SetEnableNodeLabeller - Enable/Disable the Node Labeller.
func SetEnableNodeLabeller(enable bool, builder *amdgpu.Builder, force bool) error {
	if builder.Definition.Spec.DevicePlugin.EnableNodeLabeller == nil {
		builder.Definition.Spec.DevicePlugin.EnableNodeLabeller = new(bool)
	}

	*builder.Definition.Spec.DevicePlugin.EnableNodeLabeller = enable

	_, updateErr := builder.Update(force)

	if updateErr != nil {
		return updateErr
	}

	if *GetEnableNodeLabeller(builder) != enable {
		return fmt.Errorf("mismatch in enableNodeLabeller - Expected: '%v', Got: '%v' ",
			enable, GetEnableNodeLabeller(builder))
	}

	return nil
}
