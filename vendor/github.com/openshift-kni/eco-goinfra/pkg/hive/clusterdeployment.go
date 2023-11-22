package hive

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	hiveextV1Beta1 "github.com/openshift/assisted-service/api/hiveextension/v1beta1"
	hiveV1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/openshift/hive/apis/hive/v1/agent"
	coreV1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterDeploymentBuilder provides struct for the clusterdeployment object containing connection to
// the cluster and the clusterdeployment definitions.
type ClusterDeploymentBuilder struct {
	Definition *hiveV1.ClusterDeployment
	Object     *hiveV1.ClusterDeployment
	errorMsg   string
	apiClient  *clients.Settings
}

// ClusterDeploymentAdditionalOptions additional options for ClusterDeployment object.
type ClusterDeploymentAdditionalOptions func(builder *ClusterDeploymentBuilder) (*ClusterDeploymentBuilder, error)

// NewABMClusterDeploymentBuilder creates a new instance of
// ClusterDeploymentBuilder with platform type set to agentBareMetal.
func NewABMClusterDeploymentBuilder(
	apiClient *clients.Settings,
	name string,
	nsname string,
	clusterName string,
	baseDomain string,
	clusterInstallRef string,
	agentSelector metaV1.LabelSelector) *ClusterDeploymentBuilder {
	glog.V(100).Infof(
		`Initializing new agentbaremetal clusterdeployment structure with the following params: name: %s, namespace: %s,
		  clusterName: %s, baseDomain: %s, clusterInstallRef: %s, agentSelector: %s`,
		name, nsname, clusterName, baseDomain, clusterInstallRef, agentSelector)

	builder := ClusterDeploymentBuilder{
		apiClient: apiClient,
		Definition: &hiveV1.ClusterDeployment{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
			Spec: hiveV1.ClusterDeploymentSpec{
				ClusterName: clusterName,
				BaseDomain:  baseDomain,
				ClusterInstallRef: &hiveV1.ClusterInstallLocalReference{
					Group:   hiveextV1Beta1.Group,
					Version: hiveextV1Beta1.Version,
					Kind:    "AgentClusterInstall",
					Name:    clusterInstallRef,
				},
				Platform: hiveV1.Platform{
					AgentBareMetal: &agent.BareMetalPlatform{
						AgentSelector: agentSelector,
					},
				},
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the clusterdeployment is empty")

		builder.errorMsg = "clusterdeployment 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the clusterdeployment is empty")

		builder.errorMsg = "clusterdeployment 'namespace' cannot be empty"
	}

	if clusterName == "" {
		glog.V(100).Infof("The clusterName of the clusterdeployment is empty")

		builder.errorMsg = "clusterdeployment 'clusterName' cannot be empty"
	}

	if baseDomain == "" {
		glog.V(100).Infof("The baseDomain of the clusterdeployment is empty")

		builder.errorMsg = "clusterdeployment 'baseDomain' cannot be empty"
	}

	if clusterInstallRef == "" {
		glog.V(100).Infof("The clusterInstallRef of the clusterdeployment is empty")

		builder.errorMsg = "clusterdeployment 'clusterInstallRef' cannot be empty"
	}

	return &builder
}

// WithAdditionalAgentSelectorLabels inserts additional labels
// into the clusterdeployment label selector.
func (builder *ClusterDeploymentBuilder) WithAdditionalAgentSelectorLabels(
	agentSelector map[string]string) *ClusterDeploymentBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Adding agentSelectors %s to clusterdeployment %s in namespace %s",
		agentSelector, builder.Definition.Name, builder.Definition.Namespace)

	if builder.Definition.Spec.Platform.AgentBareMetal == nil {
		glog.V(100).Infof("The clusterdeployment platform is not agentBareMetal")

		builder.errorMsg = "clusterdeployment type must be AgentBareMetal to use agentSelector"
	}

	if len(agentSelector) == 0 {
		glog.V(100).Infof("The clusterdeployment agentSelector is empty")

		builder.errorMsg = "agentSelector cannot be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if len(builder.Definition.Spec.Platform.AgentBareMetal.AgentSelector.MatchLabels) == 0 {
		builder.Definition.Spec.Platform.AgentBareMetal.AgentSelector.MatchLabels = agentSelector
	} else {
		for k, v := range agentSelector {
			builder.Definition.Spec.Platform.AgentBareMetal.AgentSelector.MatchLabels[k] = v
		}
	}

	return builder
}

// WithPullSecret adds a pull-secret reference to the clusterdeployment.
func (builder *ClusterDeploymentBuilder) WithPullSecret(psName string) *ClusterDeploymentBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Adding pull-secret ref %s to clusterdeployment %s in namespace %s",
		psName, builder.Definition.Name, builder.Definition.Namespace)

	builder.Definition.Spec.PullSecretRef = &coreV1.LocalObjectReference{Name: psName}

	return builder
}

// Get fetches the defined clusterdeployment from the cluster.
func (builder *ClusterDeploymentBuilder) Get() (*hiveV1.ClusterDeployment, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting clusterdeployment %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	clusterDeployment := &hiveV1.ClusterDeployment{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, clusterDeployment)

	if err != nil {
		return nil, err
	}

	return clusterDeployment, err
}

// PullClusterDeployment pulls existing clusterdeployment from cluster.
func PullClusterDeployment(apiClient *clients.Settings, name, nsname string) (*ClusterDeploymentBuilder, error) {
	glog.V(100).Infof("Pulling existing clusterdeployment name %s under namespace %s from cluster", name, nsname)

	builder := ClusterDeploymentBuilder{
		apiClient: apiClient,
		Definition: &hiveV1.ClusterDeployment{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the clusterdeployment is empty")

		builder.errorMsg = "clusterdeployment 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the clusterdeployment is empty")

		builder.errorMsg = "clusterdeployment 'namespace' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("clusterdeployment object %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Create generates a clusterdeployment on the cluster.
func (builder *ClusterDeploymentBuilder) Create() (*ClusterDeploymentBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the clusterdeployment %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)
		if err == nil {
			builder.Object = builder.Definition
		}
	}

	return builder, err
}

// WithOptions creates ClusterDeployment with generic mutation options.
func (builder *ClusterDeploymentBuilder) WithOptions(
	options ...ClusterDeploymentAdditionalOptions) *ClusterDeploymentBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting ClusterDeployment additional options")

	for _, option := range options {
		if option != nil {
			builder, err := option(builder)

			if err != nil {
				glog.V(100).Infof("Error occurred in mutation function")

				builder.errorMsg = err.Error()

				return builder
			}
		}
	}

	return builder
}

// Update modifies an existing clusterdeployment on the cluster.
func (builder *ClusterDeploymentBuilder) Update(force bool) (*ClusterDeploymentBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating clusterdeployment %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				msg.FailToUpdateNotification("clusterdeployment", builder.Definition.Name, builder.Definition.Namespace))

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					msg.FailToUpdateError("clusterdeployment", builder.Definition.Name, builder.Definition.Namespace))

				return nil, err
			}

			return builder.Create()
		}
	}

	if err == nil {
		builder.Object = builder.Definition
	}

	return builder, err
}

// Delete removes a clusterdeployment from the cluster.
func (builder *ClusterDeploymentBuilder) Delete() (*ClusterDeploymentBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the clusterdeployment %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return builder, fmt.Errorf("clusterdeployment cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("cannot delete clusterdeployment: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Exists checks if the defined clusterdeployment has already been created.
func (builder *ClusterDeploymentBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if clusterdeployment %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *ClusterDeploymentBuilder) validate() (bool, error) {
	resourceCRD := "ClusterDeployment"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is undefined", resourceCRD)

		builder.errorMsg = msg.UndefinedCrdObjectErrString(resourceCRD)
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiclient is nil", resourceCRD)

		builder.errorMsg = fmt.Sprintf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message: %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf(builder.errorMsg)
	}

	return true, nil
}
