package nvidiagpu

import (
	"context"
	"fmt"

	nvidiagpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Builder provides a struct for ClusterPolicy object
// from the cluster and a ClusterPolicy definition.
type Builder struct {
	// Builder definition. Used to create
	// Builder object with minimum set of required elements.
	Definition *nvidiagpuv1.ClusterPolicy
	// Created Builder object on the cluster.
	Object *nvidiagpuv1.ClusterPolicy
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// errorMsg is processed before Builder object is created.
	errorMsg string
}

// NewBuilderFromObjectString creates a Builder object from CSV alm-examples.
func NewBuilderFromObjectString(apiClient *clients.Settings, almExample string) *Builder {
	glog.V(100).Infof(
		"Initializing new Builder structure from almExample string")

	clusterPolicy, err := getClusterPolicyFromAlmExample(almExample)

	glog.V(100).Infof("Initializing Builder definition to ClusterPolicy object")

	builder := Builder{
		apiClient:  apiClient,
		Definition: clusterPolicy,
	}

	if err != nil {
		glog.V(100).Infof(
			"Error initializing ClusterPolicy from alm-examples: %s", err.Error())

		builder.errorMsg = fmt.Sprintf("Error initializing ClusterPolicy from alm-examples: %s", err.Error())
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The ClusterPolicy object definition is nil")

		builder.errorMsg = "ClusterPolicy 'Object.Definition' is nil"
	}

	return &builder
}

// Get returns clusterPolicy object if found.
func (builder *Builder) Get() (*nvidiagpuv1.ClusterPolicy, error) {
	glog.V(100).Infof(
		"Collecting ClusterPolicy object %s", builder.Definition.Name)

	clusterPolicy := &nvidiagpuv1.ClusterPolicy{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name: builder.Definition.Name,
	}, clusterPolicy)

	if err != nil {
		glog.V(100).Infof(
			"ClusterPolicy object %s doesn't exist", builder.Definition.Name)

		return nil, err
	}

	return clusterPolicy, err
}

// Pull loads an existing clusterPolicy into Builder struct.
func Pull(apiClient *clients.Settings, name string) (*Builder, error) {
	glog.V(100).Infof("Pulling existing clusterPolicy name: %s", name)

	builder := Builder{
		apiClient: apiClient,
		Definition: &nvidiagpuv1.ClusterPolicy{
			ObjectMeta: metaV1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("ClusterPolicy name is empty")

		builder.errorMsg = "ClusterPolicy 'name' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("ClusterPolicy object %s doesn't exist", name)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given ClusterPolicy exists.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof(
		"Checking if ClusterPolicy %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.Get()

	if err != nil {
		glog.V(100).Infof("Failed to collect ClusterPolicy object due to %s", err.Error())
	}

	return err == nil || !k8serrors.IsNotFound(err)
}

// Delete removes a ClusterPolicy.
func (builder *Builder) Delete() (*Builder, error) {
	glog.V(100).Infof("Deleting ClusterPolicy %s", builder.Definition.Name)

	if !builder.Exists() {
		return builder, fmt.Errorf("clusterpolicy cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("cannot delete clusterpolicy: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Create makes a ClusterPolicy in the cluster and stores the created object in struct.
func (builder *Builder) Create() (*Builder, error) {
	glog.V(100).Infof("Creating the ClusterPolicy %s", builder.Definition.Name)

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

// Update renovates the existing ClusterPolicy object with the definition in builder.
func (builder *Builder) Update(force bool) (*Builder, error) {
	glog.V(100).Infof("Updating the ClusterPolicy object named:  %s", builder.Definition.Name)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof("Failed to update the clusterpolicy object %s. "+
				"Note: Force flag set, executed delete/create methods instead", builder.Definition.Name)

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the clusterpolicy object %s."+
						"due to error in delete function", builder.Definition.Name)

				return nil, err
			}

			return builder.Create()
		}
	}

	return builder, err
}

// getClusterPolicyFromAlmExample extracts the ClusterPolicy from the alm-examples block.
func getClusterPolicyFromAlmExample(almExample string) (*nvidiagpuv1.ClusterPolicy, error) {
	clusterPolicyList := &nvidiagpuv1.ClusterPolicyList{}

	if almExample == "" {
		return nil, fmt.Errorf("almExample is an empty string")
	}

	err := json.Unmarshal([]byte(almExample), &clusterPolicyList.Items)

	if err != nil {
		return nil, err
	}

	if len(clusterPolicyList.Items) == 0 {
		return nil, fmt.Errorf("failed to get alm examples")
	}

	return &clusterPolicyList.Items[0], nil
}
