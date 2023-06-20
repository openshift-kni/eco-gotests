package hive

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	hiveV1 "github.com/openshift/hive/apis/hive/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	goclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterImageSetBuilder provides struct for the clusterimageset object containing connection to
// the cluster and the clusterimageset definitions.
type ClusterImageSetBuilder struct {
	Definition *hiveV1.ClusterImageSet
	Object     *hiveV1.ClusterImageSet
	errorMsg   string
	apiClient  *clients.Settings
}

// ClusterImageSetAdditionalOptions additional options for ClusterImageSet object.
type ClusterImageSetAdditionalOptions func(builder *ClusterImageSetBuilder) (*ClusterImageSetBuilder, error)

// NewClusterImageSetBuilder creates a new instance of ClusterImageSetBuilder.
func NewClusterImageSetBuilder(apiClient *clients.Settings, name, releaseImage string) *ClusterImageSetBuilder {
	glog.V(100).Infof(
		`Initializing new clusterimageset structure with the following params: name: %s, releaseImage: %s`,
		name, releaseImage)

	builder := ClusterImageSetBuilder{
		apiClient: apiClient,
		Definition: &hiveV1.ClusterImageSet{
			ObjectMeta: metaV1.ObjectMeta{
				Name: name,
			},
			Spec: hiveV1.ClusterImageSetSpec{
				ReleaseImage: releaseImage,
			},
		},
	}

	if apiClient == nil {
		glog.V(100).Infof("The apiClient is nil")

		builder.errorMsg = "clusterimageset cannot have nil apiClient"
	}

	if name == "" {
		glog.V(100).Infof("The name of the clusterimageset is empty")

		builder.errorMsg = "clusterimageset 'name' cannot be empty"
	}

	if releaseImage == "" {
		glog.V(100).Infof("The releaseImage of the clusterimageset is empty")

		builder.errorMsg = "clusterimageset 'releaseImage' cannot be empty"
	}

	return &builder
}

// WithReleaseImage sets the releaseImage for the clusterimageset.
func (builder *ClusterImageSetBuilder) WithReleaseImage(image string) *ClusterImageSetBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting clusterimageset %s releaseImage to %s",
		builder.Definition.Name, image)

	if image == "" {
		glog.V(100).Infof("The clusterimageset releaseImage is empty")

		builder.errorMsg = "cannot set releaseImage to empty string"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.ReleaseImage = image

	return builder
}

// WithOptions creates ClusterDeployment with generic mutation options.
func (builder *ClusterImageSetBuilder) WithOptions(
	options ...ClusterImageSetAdditionalOptions) *ClusterImageSetBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting ClusterImageSet additional options")

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

// PullClusterImageSet loads an existing clusterimageset into ClusterImageSetBuilder struct.
func PullClusterImageSet(apiClient *clients.Settings, name string) (*ClusterImageSetBuilder, error) {
	glog.V(100).Infof("Pulling existing clusterimageset name: %s", name)

	builder := ClusterImageSetBuilder{
		apiClient: apiClient,
		Definition: &hiveV1.ClusterImageSet{
			ObjectMeta: metaV1.ObjectMeta{
				Name: name,
			},
		},
	}

	if name == "" {
		builder.errorMsg = "clusterimageset 'name' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("clusterimageset object %s doesn't exist", name)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Get fetches the defined clusterimageset from the cluster.
func (builder *ClusterImageSetBuilder) Get() (*hiveV1.ClusterImageSet, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting clusterimageset %s", builder.Definition.Name)

	clusterimageset := &hiveV1.ClusterImageSet{}
	err := builder.apiClient.Get(context.TODO(), goclient.ObjectKey{
		Name: builder.Definition.Name,
	}, clusterimageset)

	if err != nil {
		return nil, err
	}

	return clusterimageset, err
}

// Create generates a clusterimageset on the cluster.
func (builder *ClusterImageSetBuilder) Create() (*ClusterImageSetBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the clusterimageset %s", builder.Definition.Name)

	var err error
	if !builder.Exists() {
		err = builder.apiClient.Create(context.TODO(), builder.Definition)
		if err == nil {
			builder.Object = builder.Definition
		}
	}

	return builder, err
}

// Update modifies an existing clusterimageset on the cluster.
func (builder *ClusterImageSetBuilder) Update(force bool) (*ClusterImageSetBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating clusterimageset %s", builder.Definition.Name)

	err := builder.apiClient.Update(context.TODO(), builder.Definition)

	if err != nil {
		if force {
			glog.V(100).Infof(
				"Failed to update the clusterimageset object %s. "+
					"Note: Force flag set, executed delete/create methods instead",
				builder.Definition.Name,
			)

			builder, err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					"Failed to update the clusterimageset object %s, "+
						"due to error in delete function", builder.Definition.Name,
				)

				return nil, err
			}

			return builder.Create()
		}
	}

	return builder, err
}

// Delete removes a clusterimageset from the cluster.
func (builder *ClusterImageSetBuilder) Delete() (*ClusterImageSetBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the clusterimageset %s", builder.Definition.Name)

	if !builder.Exists() {
		return builder, fmt.Errorf("clusterimageset cannot be deleted because it does not exist")
	}

	err := builder.apiClient.Delete(context.TODO(), builder.Definition)

	if err != nil {
		return builder, fmt.Errorf("cannot delete clusterimageset: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Exists checks if the defined clusterimageset has already been created.
func (builder *ClusterImageSetBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if clusterimageset %s exists", builder.Definition.Name)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *ClusterImageSetBuilder) validate() (bool, error) {
	resourceCRD := "ClusterImageSet"

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
