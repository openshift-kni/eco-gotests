package hive

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/eco-gotests/pkg/msg"
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
	glog.V(100).Infof(
		"Adding agentSelectors %s to clusterdeployment %s in namespace %s",
		agentSelector, builder.Definition.Name, builder.Definition.Namespace)

	if builder.Definition == nil {
		glog.V(100).Infof("The clusterdeployment is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("ClusterDeployment")
	}

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
	glog.V(100).Infof(
		"Adding pull-secret ref %s to clusterdeployment %s in namespace %s",
		psName, builder.Definition.Name, builder.Definition.Namespace)

	if builder.Definition == nil {
		glog.V(100).Infof("The clusterdeployment is undefined")

		builder.errorMsg = msg.UndefinedCrdObjectErrString("ClusterDeployment")
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.PullSecretRef = &coreV1.LocalObjectReference{Name: psName}

	return builder
}

// Get fetches the defined clusterdeployment from the cluster.
func (builder *ClusterDeploymentBuilder) Get() (*hiveV1.ClusterDeployment, error) {
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

// ListClusterDeploymentsInAllNamespaces returns a cluster-wide clusterdeployment inventory.
func ListClusterDeploymentsInAllNamespaces(
	apiClient *clients.Settings,
	options goclient.ListOption) ([]*ClusterDeploymentBuilder, error) {
	glog.V(100).Infof("Listing all clusterdeployments with the options %v", options)

	clusterDeployments := new(hiveV1.ClusterDeploymentList)
	err := apiClient.List(context.TODO(), clusterDeployments, options)

	if err != nil {
		glog.V(100).Infof("Failed to list all clusterDeployments due to %s", err.Error())

		return nil, err
	}

	var clusterDeploymentObjects []*ClusterDeploymentBuilder

	for _, clusterDeployment := range clusterDeployments.Items {
		copiedClusterDeployment := clusterDeployment
		clusterDeploymentBuilder := &ClusterDeploymentBuilder{
			apiClient:  apiClient,
			Object:     &copiedClusterDeployment,
			Definition: &copiedClusterDeployment,
		}

		clusterDeploymentObjects = append(clusterDeploymentObjects, clusterDeploymentBuilder)
	}

	return clusterDeploymentObjects, nil
}

// Create generates a clusterdeployment on the cluster.
func (builder *ClusterDeploymentBuilder) Create() (*ClusterDeploymentBuilder, error) {
	glog.V(100).Infof("Creating the clusterdeployment %s in namespace %s",
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

// Update modifies an existing clusterdeployment on the cluster.
func (builder *ClusterDeploymentBuilder) Update(force bool) (*ClusterDeploymentBuilder, error) {
	glog.V(100).Infof("Updating clusterdeployment %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				"Failed to update the clusterdeployment object %s in namespace %s. "+
					"Note: Force flag set, executed delete/create methods instead",
				builder.Definition.Name, builder.Definition.Namespace,
			)

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the clusterdeployment object %s in namespace %s, "+
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

// Delete removes a clusterdeployment from the cluster.
func (builder *ClusterDeploymentBuilder) Delete() (*ClusterDeploymentBuilder, error) {
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
	glog.V(100).Infof("Checking if clusterdeployment %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}
