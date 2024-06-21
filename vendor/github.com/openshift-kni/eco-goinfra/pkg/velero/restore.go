package velero

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroClient "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RestoreBuilder provides a struct for restore object from the cluster and a restore definition.
type RestoreBuilder struct {
	// Restore definition, used to create the restore object.
	Definition *velerov1.Restore
	// Created restore object.
	Object *velerov1.Restore
	// Used to store latest error message upon defining or mutating restore definition.
	errorMsg string
	// api client to interact with the cluster.
	apiClient veleroClient.Interface
}

// NewRestoreBuilder creates a new instance of RestoreBuilder.
func NewRestoreBuilder(apiClient *clients.Settings, name, nsname, backupName string) *RestoreBuilder {
	glog.V(100).Infof(
		"Initializing new restore structure with the following params: "+
			"name: %s, namespace: %s, restoreName: %s", name, nsname, backupName)

	builder := &RestoreBuilder{
		apiClient: apiClient.VeleroClient,
		Definition: &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
			Spec: velerov1.RestoreSpec{
				BackupName: backupName,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the restore is empty")

		builder.errorMsg = "restore name cannot be an empty string"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the restore is empty")

		builder.errorMsg = "restore namespace cannot be an empty string"
	}

	if backupName == "" {
		glog.V(100).Infof("The backupName of the restore is empty")

		builder.errorMsg = "restore backupName cannot be an empty string"
	}

	return builder
}

// PullRestore loads an existing restore into RestoreBuilder struct.
func PullRestore(apiClient *clients.Settings, name, nsname string) (*RestoreBuilder, error) {
	glog.V(100).Infof("Pulling existing restore name: %s under namespace: %s", name, nsname)

	builder := RestoreBuilder{
		apiClient: apiClient.VeleroClient,
		Definition: &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		return nil, fmt.Errorf("restore name cannot be empty")
	}

	if nsname == "" {
		return nil, fmt.Errorf("restore namespace cannot be empty")
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("restore object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// WithStorageLocation adds a storage location to the restore.
func (builder *RestoreBuilder) WithStorageLocation(location string) *RestoreBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Adding storage location %s to restore %s in namespace %s",
		location, builder.Definition.Name, builder.Definition.Namespace)

	if location == "" {
		glog.V(100).Infof("Backup storage location is empty")

		builder.errorMsg = "restore storage location cannot be an empty string"
	}

	if builder.errorMsg != "" {
		return builder
	}

	if builder.Definition.ObjectMeta.Labels == nil {
		builder.Definition.ObjectMeta.Labels = make(map[string]string)
	}

	builder.Definition.ObjectMeta.Labels["velero.io/storage-location"] = location

	return builder
}

// Exists checks whether the given restore object exists.
func (builder *RestoreBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if restore %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.VeleroV1().Restores(builder.Definition.Namespace).Get(
		context.TODO(), builder.Definition.Name, metav1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// Create makes a restore according to the restore definition and stores the created object in the restore builder.
func (builder *RestoreBuilder) Create() (*RestoreBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating restore %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.VeleroV1().Restores(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metav1.CreateOptions{})
	}

	return builder, err
}

// Update renovates the existing restore object with the restore definition in builder.
func (builder *RestoreBuilder) Update() (*RestoreBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating restore %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.VeleroV1().Restores(builder.Definition.Namespace).Update(
		context.TODO(), builder.Definition, metav1.UpdateOptions{})

	return builder, err
}

// Delete removes the restore object and resets the builder object.
func (builder *RestoreBuilder) Delete() (*RestoreBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting restore %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return builder, fmt.Errorf("restore cannot be deleted because it does not exist")
	}

	err := builder.apiClient.VeleroV1().Restores(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metav1.DeleteOptions{})

	if err != nil {
		return builder, fmt.Errorf("can not delete restore: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *RestoreBuilder) validate() (bool, error) {
	resourceCRD := "Restore"

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
