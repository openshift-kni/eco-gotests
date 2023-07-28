package pod

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"
)

var (
	// AllowedSCList list of allowed SecurityCapabilities.
	AllowedSCList          = []string{"NET_RAW", "NET_ADMIN", "SYS_ADMIN", "ALL"}
	falseVar               = false
	trueVar                = true
	capabilityAll          = []v1.Capability{"ALL"}
	defaultGroupID         = int64(3000)
	defaultUserID          = int64(2000)
	defaultSecurityContext = &v1.SecurityContext{
		AllowPrivilegeEscalation: &falseVar,
		RunAsNonRoot:             &trueVar,
		SeccompProfile:           &v1.SeccompProfile{Type: "RuntimeDefault"},
		Capabilities: &v1.Capabilities{
			Drop: capabilityAll,
		},
		RunAsGroup: &defaultGroupID,
		RunAsUser:  &defaultUserID,
	}
)

// ContainerBuilder provides a struct for container's object definition.
type ContainerBuilder struct {
	// Container definition, used to create the Container object.
	definition *v1.Container
	// Used to store latest error message upon defining or mutating container definition.
	errorMsg string
}

// NewContainerBuilder creates a new instance of ContainerBuilder.
func NewContainerBuilder(name, image string, cmd []string) *ContainerBuilder {
	glog.V(100).Infof("Initializing new container structure with the following params: "+
		"name: %s, image: %s, cmd: %v", name, image, cmd)

	builder := &ContainerBuilder{
		definition: &v1.Container{
			Name:            name,
			Image:           image,
			Command:         cmd,
			SecurityContext: defaultSecurityContext,
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the container is empty")

		builder.errorMsg = "container's name is empty"
	}

	if image == "" {
		glog.V(100).Infof("Container's image is empty")

		builder.errorMsg = "container's image is empty"
	}

	if len(cmd) < 1 {
		glog.V(100).Infof("Container's cmd is empty")

		builder.errorMsg = "container's cmd is empty"
	}

	return builder
}

// WithSecurityCapabilities applies SecurityCapabilities to the container definition.
func (builder *ContainerBuilder) WithSecurityCapabilities(sCapabilities []string, redefine bool) *ContainerBuilder {
	glog.V(100).Infof("Applying a list of SecurityCapabilities %v to container %s",
		sCapabilities, builder.definition.Name)

	if builder.definition.SecurityContext != nil {
		if !redefine {
			glog.V(100).Infof("Cannot modify pre-existing SecurityContext")

			builder.errorMsg = "can not modify pre-existing security context"
		}

		builder.definition.SecurityContext = nil
	}

	if !areCapabilitiesValid(sCapabilities) {
		glog.V(100).Infof("Given SecurityCapabilities %v are not valid. Valid list %s",
			sCapabilities, AllowedSCList)

		builder.errorMsg = "one of the give securityCapabilities is invalid. Please extend allowed list or fix parameter"
	}

	if builder.errorMsg != "" {
		return builder
	}

	var sCapabilitiesList []v1.Capability
	for _, capability := range sCapabilities {
		sCapabilitiesList = append(sCapabilitiesList, v1.Capability(capability))
	}

	builder.definition.SecurityContext = &v1.SecurityContext{
		Capabilities: &v1.Capabilities{
			Add: sCapabilitiesList,
		},
	}

	return builder
}

// WithSecurityContext applies security Context on container.
func (builder *ContainerBuilder) WithSecurityContext(securityContext *v1.SecurityContext) *ContainerBuilder {
	glog.V(100).Infof("Applying custom securityContext %v", securityContext)

	if securityContext == nil {
		glog.V(100).Infof("Cannot add empty securityContext to container structure")

		builder.errorMsg = "can not modify container config with empty securityContext"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.definition.SecurityContext = securityContext

	return builder
}

// WithResourceLimit applies resource limit on container.
func (builder *ContainerBuilder) WithResourceLimit(hugePages, memory string, cpu int64) *ContainerBuilder {
	glog.V(100).Infof("Applying custom resource limit to container: hugePages: %s memory: %s cpu: %d",
		hugePages, memory, cpu)

	if hugePages == "" {
		glog.V(100).Infof("Container's resource limit hugePages is empty")

		builder.errorMsg = "container's resource limit 'hugePages' is empty"
	}

	if memory == "" {
		glog.V(100).Infof("Container's resource limit memory is empty")

		builder.errorMsg = "container's resource limit 'memory' is empty"
	}

	if cpu <= 0 {
		glog.V(100).Infof("Container's resource limit cpu can not be zero or negative number.")

		builder.errorMsg = "container's resource limit 'cpu' is invalid"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.definition.Resources.Limits = v1.ResourceList{
		"hugepages-1Gi": resource.MustParse(hugePages),
		"memory":        resource.MustParse(memory),
		"cpu":           *resource.NewQuantity(cpu, resource.DecimalSI),
	}

	return builder
}

// WithResourceRequest applies resource request on container.
func (builder *ContainerBuilder) WithResourceRequest(hugePages, memory string, cpu int64) *ContainerBuilder {
	glog.V(100).Infof("Applying custom resource request to container: hugePages: %s memory: %s cpu: %d",
		hugePages, memory, cpu)

	if hugePages == "" {
		glog.V(100).Infof("Container's resource request hugePages is empty")

		builder.errorMsg = "container's resource request 'hugePages' is empty"
	}

	if memory == "" {
		glog.V(100).Infof("Container's resource request memory is empty")

		builder.errorMsg = "container's resource request 'memory' is empty"
	}

	if cpu <= 0 {
		glog.V(100).Infof("Container's resource request cpu can not be zero or negative number.")

		builder.errorMsg = "container's resource request 'cpu' is invalid"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.definition.Resources.Requests = v1.ResourceList{
		"hugepages-1Gi": resource.MustParse(hugePages),
		"memory":        resource.MustParse(memory),
		"cpu":           *resource.NewQuantity(int64(2), resource.DecimalSI),
	}

	return builder
}

// WithEnvVar adds environment variables to container.
func (builder *ContainerBuilder) WithEnvVar(name, value string) *ContainerBuilder {
	glog.V(100).Infof("Applying custom environment variables to container: name %s, value: %s", name, value)

	if name == "" {
		glog.V(100).Infof("Container's environment var 'name' is empty")

		builder.errorMsg = "container's environment var 'name' is empty"
	}

	if value == "" {
		glog.V(100).Infof("Container's environment var 'value' is empty")

		builder.errorMsg = "container's environment var 'value' is empty"
	}

	if builder.errorMsg != name {
		return builder
	}

	if builder.definition.Env != nil {
		builder.definition.Env = append(builder.definition.Env, v1.EnvVar{Name: name, Value: value})

		return builder
	}

	builder.definition.Env = []v1.EnvVar{{Name: name, Value: value}}

	return builder
}

// GetContainerCfg returns Container struct.
func (builder *ContainerBuilder) GetContainerCfg() (*v1.Container, error) {
	glog.V(100).Infof("Returning configuration for container %s", builder.definition.Name)

	if builder.errorMsg != "" {
		glog.V(100).Infof("Failed to build container configuration due to %s", builder.errorMsg)

		return nil, fmt.Errorf(builder.errorMsg)
	}

	return builder.definition, nil
}

func areCapabilitiesValid(capabilities []string) bool {
	valid := true

	for _, capability := range capabilities {
		if !slices.Contains(AllowedSCList, capability) {
			valid = false
		}
	}

	return valid
}
