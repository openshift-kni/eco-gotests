package assisted

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/msg"
	agentInstallV1Beta1 "github.com/openshift/assisted-service/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// InfraEnvBuilder provides struct for the infraenv object containing connection to
// the cluster and the infraenv definitions.
type InfraEnvBuilder struct {
	Definition *agentInstallV1Beta1.InfraEnv
	Object     *agentInstallV1Beta1.InfraEnv
	errorMsg   string
	apiClient  *clients.Settings
}

// NewInfraEnvBuilder creates a new instance of InfraEnvBuilder.
func NewInfraEnvBuilder(apiClient *clients.Settings, name, nsname, psName string) *InfraEnvBuilder {
	glog.V(100).Infof(
		"Initializing new infraenv structure with the following params: "+
			"name: %s, namespace: %s, pull-secret: %s",
		name, nsname, psName)

	builder := InfraEnvBuilder{
		apiClient: apiClient,
		Definition: &agentInstallV1Beta1.InfraEnv{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
			Spec: agentInstallV1Beta1.InfraEnvSpec{
				PullSecretRef: &corev1.LocalObjectReference{
					Name: psName,
				},
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the infraenv is empty")

		builder.errorMsg = "infraenv 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the infraenv is empty")

		builder.errorMsg = "infraenv 'namespace' cannot be empty"
	}

	if psName == "" {
		glog.V(100).Infof("The pull-secret ref of the infraenv is empty")

		builder.errorMsg = "infraenv 'pull-secret' cannot be empty"
	}

	return &builder
}

// WithClusterRef sets the cluster reference to be used by the infraenv.
func (builder *InfraEnvBuilder) WithClusterRef(name, nsname string) *InfraEnvBuilder {
	glog.V(100).Infof("Adding clusterRef %s in namespace %s to InfraEnv %s", name, nsname, builder.Definition.Name)

	if builder.Definition == nil {
		glog.V(100).Infof("The infraenv is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("InfraEnv")
	}

	if name == "" {
		glog.V(100).Infof("The name of the infraenv clusterRef is empty")

		builder.errorMsg = "infraenv clusterRef 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the infraenv clusterRef is empty")

		builder.errorMsg = "infraenv clusterRef 'namespace' cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.ClusterRef = &agentInstallV1Beta1.ClusterReference{
		Name:      name,
		Namespace: nsname,
	}

	return builder
}

// WithAdditionalNTPSource adds additional servers as NTP sources for the spoke cluster.
func (builder *InfraEnvBuilder) WithAdditionalNTPSource(ntpSource string) *InfraEnvBuilder {
	glog.V(100).Infof("Adding ntpSource %s to InfraEnv %s", ntpSource, builder.Definition.Name)

	if builder.Definition == nil {
		glog.V(100).Infof("The infraenv is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("InfraEnv")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.AdditionalNTPSources = append(builder.Definition.Spec.AdditionalNTPSources, ntpSource)

	return builder
}

// WithSSHAuthorizedKey sets the authorized ssh key for accessing the nodes during discovery.
func (builder *InfraEnvBuilder) WithSSHAuthorizedKey(sshAuthKey string) *InfraEnvBuilder {
	glog.V(100).Infof("Adding sshAuthorizedKey %s to InfraEnv %s", sshAuthKey, builder.Definition.Name)

	if builder.Definition == nil {
		glog.V(100).Infof("The infraenv is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("InfraEnv")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.SSHAuthorizedKey = sshAuthKey

	return builder
}

// WithAgentLabel adds labels to be applied to agents that boot from the infraenv.
func (builder *InfraEnvBuilder) WithAgentLabel(key, value string) *InfraEnvBuilder {
	glog.V(100).Infof("Adding agentLabel %s:%s to InfraEnv %s", key, value, builder.Definition.Name)

	if builder.Definition == nil {
		glog.V(100).Infof("The infraenv is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("InfraEnv")
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.Spec.AgentLabels == nil {
		builder.Definition.Spec.AgentLabels = make(map[string]string)
	}

	builder.Definition.Spec.AgentLabels[key] = value

	return builder
}

// WithProxy includes a proxy configuration to be used by the infraenv.
func (builder *InfraEnvBuilder) WithProxy(proxy agentInstallV1Beta1.Proxy) *InfraEnvBuilder {
	glog.V(100).Infof("Adding proxy %s to InfraEnv %s", proxy, builder.Definition.Name)

	if builder.Definition == nil {
		glog.V(100).Infof("The infraenv is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("InfraEnv")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Proxy = &proxy

	return builder
}

// WithNmstateConfigLabelSelector adds a selector for identifying
// nmstateconfigs that should be applied to this infraenv.
func (builder *InfraEnvBuilder) WithNmstateConfigLabelSelector(selector metaV1.LabelSelector) *InfraEnvBuilder {
	glog.V(100).Infof("Adding nmstateconfig selector %s to InfraEnv %s", &selector, builder.Definition.Name)

	if builder.Definition == nil {
		glog.V(100).Infof("The infraenv is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("InfraEnv")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.NMStateConfigLabelSelector = selector

	return builder
}

// WithCPUType sets the cpu architecture for the discovery ISO.
func (builder *InfraEnvBuilder) WithCPUType(arch string) *InfraEnvBuilder {
	glog.V(100).Infof("Adding cpuArchitecture %s to InfraEnv %s", arch, builder.Definition.Name)

	if builder.Definition == nil {
		glog.V(100).Infof("The infraenv is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("InfraEnv")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.CpuArchitecture = arch

	return builder
}

// WithIgnitionConfigOverride includes the specified ignitionconfigoverride for discovery.
func (builder *InfraEnvBuilder) WithIgnitionConfigOverride(override string) *InfraEnvBuilder {
	glog.V(100).Infof("Adding ignitionConfigOverride %s to InfraEnv %s", override, builder.Definition.Name)

	if builder.Definition == nil {
		glog.V(100).Infof("The infraenv is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("InfraEnv")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.IgnitionConfigOverride = override

	return builder
}

// WithIPXEScriptType modifies the IPXE script type generated by the infraenv.
func (builder *InfraEnvBuilder) WithIPXEScriptType(scriptType agentInstallV1Beta1.IPXEScriptType) *InfraEnvBuilder {
	glog.V(100).Infof("Adding ipxeScriptType %s to InfraEnv %s", scriptType, builder.Definition.Name)

	if builder.Definition == nil {
		glog.V(100).Infof("The infraenv is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("InfraEnv")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.IPXEScriptType = scriptType

	return builder
}

// WithKernelArgument appends kernel configurations to be configured by the infraenv.
func (builder *InfraEnvBuilder) WithKernelArgument(kernelArg agentInstallV1Beta1.KernelArgument) *InfraEnvBuilder {
	glog.V(100).Infof("Adding kernelArgument %s to InfraEnv %s", kernelArg, builder.Definition.Name)

	if builder.Definition == nil {
		glog.V(100).Infof("The infraenv is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("InfraEnv")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.KernelArguments = append(builder.Definition.Spec.KernelArguments, kernelArg)

	return builder
}

// WaitForDiscoveryISOCreation waits the defined timeout for the discovery ISO to be generated.
func (builder *InfraEnvBuilder) WaitForDiscoveryISOCreation(timeout time.Duration) (*InfraEnvBuilder, error) {
	if builder.Definition == nil {
		glog.V(100).Infof("The infraenv is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("InfraEnv")
	}

	if builder.errorMsg != "" {
		return builder, nil
	}

	// Polls every second to determine if infraenv in desired state.
	var err error
	err = wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		builder.Object, err = builder.Get()

		if err != nil {
			return false, nil
		}

		return builder.Object.Status.CreatedTime != nil, nil

	})

	if err == nil {
		return builder, nil
	}

	return nil, err
}

// Get fetches the defined infraenv from the cluster.
func (builder *InfraEnvBuilder) Get() (*agentInstallV1Beta1.InfraEnv, error) {
	glog.V(100).Infof("Getting infraenv %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	infraEnv := &agentInstallV1Beta1.InfraEnv{}

	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, infraEnv)

	if err != nil {
		return nil, err
	}

	return infraEnv, err
}

// Create generates a infraenv on the cluster.
func (builder *InfraEnvBuilder) Create() (*InfraEnvBuilder, error) {
	glog.V(100).Infof("Creating the infraenv %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)
		if err == nil {
			builder.Object = builder.Definition
		}
	}

	return builder, err
}

// Update modifies an existing infraenv on the cluster.
func (builder *InfraEnvBuilder) Update(force bool) (*InfraEnvBuilder, error) {
	glog.V(100).Infof("Updating infraenv %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		glog.V(100).Infof("infraenv %s in namespace %s does not exist",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.errorMsg = "Cannot update non-existent infraenv"
	}

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				"Failed to update the infraenv object %s in namespace %s. "+
					"Note: Force flag set, executed delete/create methods instead",
				builder.Definition.Name, builder.Definition.Namespace,
			)

			_, err = builder.Delete()
			builder.Definition.ResourceVersion = ""

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the infraenv object %s in namespace %s, "+
						"due to error in delete function",
					builder.Definition.Name, builder.Definition.Namespace,
				)

				return nil, err
			}

			return builder.Create()
		}
	}

	return builder, err
}

// Delete removes an infraenv from the cluster.
func (builder *InfraEnvBuilder) Delete() (*InfraEnvBuilder, error) {
	glog.V(100).Infof("Deleting the infraenv %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return builder, fmt.Errorf("infraenv cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("cannot delete infraenv: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Exists checks if the defined infraenv has already been created.
func (builder *InfraEnvBuilder) Exists() bool {
	glog.V(100).Infof("Checking if infraenv %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}
