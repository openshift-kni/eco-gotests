package oadp

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	"github.com/openshift-kni/eco-goinfra/pkg/oadp/oadptypes"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DPABuilder provides a struct for backup object from the cluster and a backup definition.
type DPABuilder struct {
	// Backup definition, used to create the backup object.
	Definition *oadptypes.DataProtectionApplication
	// Created backup object.
	Object *oadptypes.DataProtectionApplication
	// Used to store latest error message upon defining or mutating backup definition.
	errorMsg string
	// api client to interact with the cluster.
	apiClient *clients.Settings
}

// GetDataProtectionApplicationGVR returns dataprotectionapplication's GroupVersionResource.
func GetDataProtectionApplicationGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group: APIGroup, Version: V1Alpha1Version, Resource: "dataprotectionapplications",
	}
}

// NewDPABuilder creates a new instance of DPABuilder.
func NewDPABuilder(
	apiClient *clients.Settings, name, namespace string, config oadptypes.ApplicationConfig) *DPABuilder {
	glog.V(100).Infof(
		"Initializing new dataprotectionapplication structure with the following params: "+
			"name: %s, namespace: %s, config %v",
		name, namespace, config, config)

	if apiClient == nil {
		glog.V(100).Infof("apiClient is nil")

		return nil
	}

	builder := &DPABuilder{
		apiClient: apiClient,
		Definition: &oadptypes.DataProtectionApplication{
			TypeMeta: metav1.TypeMeta{
				Kind:       DPAKind,
				APIVersion: V1Alpha1Version,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: oadptypes.DataProtectionApplicationSpec{
				Configuration: &config,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the dataprotectionapplication is empty")

		builder.errorMsg = "dataprotectionapplication 'name' cannot be empty"
	}

	if namespace == "" {
		glog.V(100).Infof("The namespace of the dataprotectionapplication is empty")

		builder.errorMsg = "dataprotectionapplication 'namespace' cannot be empty"
	}

	if config.Velero == nil {
		glog.V(100).Infof("The velero config of the dataprotectionapplication is empty")

		builder.errorMsg = "dataprotectionapplication velero config cannot be empty"
	}

	return builder
}

// PullDPA pulls existing dataprotectionapplication from cluster.
func PullDPA(apiClient *clients.Settings, name, nsname string) (*DPABuilder, error) {
	glog.V(100).Infof("Pulling existing dataprotectionapplication name %s under namespace %s from cluster", name, nsname)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient cannot be nil")

		return nil, fmt.Errorf("the apiClient is nil")
	}

	builder := DPABuilder{
		apiClient: apiClient,
		Definition: &oadptypes.DataProtectionApplication{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the dataprotectionapplication is empty")

		return nil, fmt.Errorf("dataprotectionapplication 'name' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the dataprotectionapplication is empty")

		return nil, fmt.Errorf("dataprotectionapplication 'namespace' cannot be empty")
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("dataprotectionapplication object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// WithBackupLocation configures the dataprotectionapplication with the specified backup location.
func (builder *DPABuilder) WithBackupLocation(backupLocation oadptypes.BackupLocation) *DPABuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Adding backuplocation to dataprotectionapplication %s in namespace %s: %v",
		builder.Definition.Name, builder.Definition.Namespace, backupLocation)

	if backupLocation.Velero == nil {
		glog.V(100).Infof("The backuplocation velero config of the dataprotectionapplication is empty")

		builder.errorMsg = "dataprotectionapplication backuplocation cannot have empty velero config"

		return builder
	}

	builder.Definition.Spec.BackupLocations = append(builder.Definition.Spec.BackupLocations, backupLocation)

	return builder
}

// Get fetches the defined dataprotectionapplication from the cluster.
func (builder *DPABuilder) Get() (*oadptypes.DataProtectionApplication, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting dataprotectionapplication %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	unsObject, err := builder.apiClient.Resource(GetDataProtectionApplicationGVR()).Namespace(
		builder.Definition.Namespace).Get(context.TODO(), builder.Definition.Name, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return builder.convertToStructured(unsObject)
}

// Exists checks whether the given dataprotectionapplication exists.
func (builder *DPABuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if dataprotectionapplication %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Create makes a dataprotectionapplication according to the dataprotectionapplication
// definition and stores the created object in the dataprotectionapplication builder.
func (builder *DPABuilder) Create() (*DPABuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the dataprotectionapplication %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	if !builder.Exists() {
		unstructuredDPA, err := runtime.DefaultUnstructuredConverter.ToUnstructured(builder.Definition)

		if err != nil {
			glog.V(100).Infof("Failed to convert structured DataProtectionApplication to unstructured object")

			return nil, err
		}

		unsObject, err := builder.apiClient.Resource(
			GetDataProtectionApplicationGVR()).Namespace(builder.Definition.Namespace).Create(
			context.TODO(), &unstructured.Unstructured{Object: unstructuredDPA}, metav1.CreateOptions{})

		if err != nil {
			glog.V(100).Infof("Failed to create dataprotectionapplication")

			return nil, err
		}

		builder.Object, err = builder.convertToStructured(unsObject)

		if err != nil {
			return nil, err
		}
	}

	return builder, err
}

// Update renovates the existing dataprotectionapplication object with
// the dataprotectionapplication definition in builder.
func (builder *DPABuilder) Update(force bool) (*DPABuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("failed to update dataprotectionapplication, object does not exist on cluster")
	}

	glog.V(100).Infof("Updating the dataprotectionapplication object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	builder.Definition.ResourceVersion = builder.Object.ResourceVersion
	builder.Definition.ObjectMeta.ResourceVersion = builder.Object.ObjectMeta.ResourceVersion

	unstructuredDPA, err := runtime.DefaultUnstructuredConverter.ToUnstructured(builder.Definition)
	if err != nil {
		glog.V(100).Infof("Failed to convert structured DataProtectionApplication to unstructured object")

		return nil, err
	}

	unstructObj, err := builder.apiClient.Resource(
		GetDataProtectionApplicationGVR()).Namespace(builder.Definition.Namespace).Update(
		context.TODO(), &unstructured.Unstructured{Object: unstructuredDPA}, metav1.UpdateOptions{})

	if err != nil {
		if force {
			glog.V(100).Infof(
				msg.FailToUpdateNotification("dataprotectionapplication", builder.Definition.Name, builder.Definition.Namespace))

			err := builder.Delete()

			if err != nil {
				glog.V(100).Infof(
					msg.FailToUpdateError("dataprotectionapplication", builder.Definition.Name, builder.Definition.Namespace))

				return nil, err
			}

			return builder.Create()
		}
	}

	if err == nil {
		structuredDPA, err := builder.convertToStructured(unstructObj)
		if err != nil {
			glog.V(100).Infof("Failed to convert unstructured dataprotectionapplication into structured object")

			return nil, err
		}

		builder.Object = structuredDPA
	}

	return builder, err
}

// Delete removes the dataprotectionapplication object and resets the builder object.
func (builder *DPABuilder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the dataprotectionapplication object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace,
	)

	if !builder.Exists() {
		glog.V(100).Infof("Dataprotectionapplication %s in namespace %s cannot be deleted because it does not exist",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil
	}

	err := builder.apiClient.Resource(
		GetDataProtectionApplicationGVR()).Namespace(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Definition.Name, metav1.DeleteOptions{})

	if err != nil {
		return fmt.Errorf("can not delete dataprotectionapplication: %w", err)
	}

	builder.Object = nil

	return nil
}

func (builder *DPABuilder) convertToStructured(
	unsObject *unstructured.Unstructured) (*oadptypes.DataProtectionApplication, error) {
	dpaBuilder := &oadptypes.DataProtectionApplication{}

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unsObject.Object, dpaBuilder)
	if err != nil {
		glog.V(100).Infof(
			"Failed to convert from unstructured to DataProtectionApplication object in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return dpaBuilder, err
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *DPABuilder) validate() (bool, error) {
	resourceCRD := "DataProtectionApplication"

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
