package argocd

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/argocd/argocdtypes"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// APIGroup const definition.
	APIGroup = "argoproj.io"
	// APIVersion const definition.
	APIVersion = "v1alpha1"
)

// ApplicationBuilder provides a struct for an application object from the cluster and a definition.
type ApplicationBuilder struct {
	// application Definition, used to create the application object.
	Definition *argocdtypes.Application
	// created application object.
	Object *argocdtypes.Application
	// api client to interact with the cluster.
	apiClient *clients.Settings
	// used to store latest error message upon defining or mutating application definition.
	errorMsg string
}

// PullApplication pulls existing application into ApplicationBuilder struct.
func PullApplication(apiClient *clients.Settings, name, nsname string) (*ApplicationBuilder, error) {
	glog.V(100).Infof("Pulling existing Application name %s under namespace %s from cluster", name, nsname)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient is empty")

		return nil, fmt.Errorf("application 'apiClient' cannot be empty")
	}

	builder := ApplicationBuilder{
		apiClient: apiClient,
		Definition: &argocdtypes.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the Application is empty")

		return nil, fmt.Errorf("application 'name' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the Application is empty")

		return nil, fmt.Errorf("application 'namespace' cannot be empty")
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("application object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Exists checks whether the given argocd application exists.
func (builder *ApplicationBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if argocd app %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// Get returns argocd application object if found.
func (builder *ApplicationBuilder) Get() (*argocdtypes.Application, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting argocd app %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	unsObject, err := builder.apiClient.Resource(
		GetApplicationsGVR()).Namespace(builder.Definition.Namespace).Get(
		context.TODO(), builder.Definition.Name, metav1.GetOptions{})

	if err != nil {
		glog.V(100).Infof(
			"Failed to Get Application object in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return builder.convertToStructured(unsObject)
}

// Update renovates the existing argocd application object with the argocd application definition in builder.
func (builder *ApplicationBuilder) Update(force bool) (*ApplicationBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Updating the argocd application object %s in namespace %s", builder.Definition.Name,
		builder.Definition.Namespace)

	unstructuredApplication, err := runtime.DefaultUnstructuredConverter.ToUnstructured(builder.Definition)

	if err != nil {
		glog.V(100).Infof("Failed to convert structured Application to unstructured object")

		return nil, err
	}

	_, err = builder.apiClient.Resource(
		GetApplicationsGVR()).Namespace(builder.Definition.Namespace).Update(
		context.TODO(), &unstructured.Unstructured{Object: unstructuredApplication}, metav1.UpdateOptions{})

	if err != nil {
		if force {
			glog.V(100).Infof(
				msg.FailToUpdateNotification("Application", builder.Definition.Name, builder.Definition.Namespace))

			builder, err := builder.Delete()
			builder.Definition.ResourceVersion = ""

			if err != nil {
				glog.V(100).Infof(
					msg.FailToUpdateError("Application", builder.Definition.Name, builder.Definition.Namespace))

				return nil, err
			}

			return builder.Create()
		}
	}

	builder.Object = builder.Definition

	return builder, err
}

// Delete removes the argocd application object from a cluster.
func (builder *ApplicationBuilder) Delete() (*ApplicationBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Deleting the argocd application object %s from namespace: %s", builder.Definition.Name,
		builder.Definition.Namespace)

	if !builder.Exists() {
		glog.V(100).Infof("application %s in namespace %s cannot be deleted because it does not exist",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.Object = nil

		return builder, nil
	}

	err := builder.apiClient.Resource(
		GetApplicationsGVR()).Namespace(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Definition.Name, metav1.DeleteOptions{})

	if err != nil {
		return builder, fmt.Errorf("can not delete argocd application: %w", err)
	}

	builder.Object = nil

	return builder, nil
}

// Create makes an argocd application in the cluster and stores the created object in a struct.
func (builder *ApplicationBuilder) Create() (*ApplicationBuilder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating argocd application %s in namespace: %s", builder.Definition.Name,
		builder.Definition.Namespace)

	var err error
	if !builder.Exists() {
		unstructuredApplication, err := runtime.DefaultUnstructuredConverter.ToUnstructured(builder.Definition)

		if err != nil {
			glog.V(100).Infof("Failed to convert structured Application to unstructured object")

			return nil, err
		}

		unsObject, err := builder.apiClient.Resource(
			GetApplicationsGVR()).Namespace(builder.Definition.Namespace).Create(
			context.TODO(), &unstructured.Unstructured{Object: unstructuredApplication}, metav1.CreateOptions{})

		if err != nil {
			glog.V(100).Infof("Failed to create Application")

			return nil, err
		}

		builder.Object, err = builder.convertToStructured(unsObject)

		if err != nil {
			return nil, err
		}
	}

	return builder, err
}

// WithGitDetails applies git details to application definition.
func (builder *ApplicationBuilder) WithGitDetails(gitRepo, gitBranch, gitPath string) *ApplicationBuilder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	if gitRepo == "" {
		glog.V(100).Infof("The 'gitRepo' of the argocd application is empty")

		builder.errorMsg = "'gitRepo' parameter is empty"
	}

	if gitBranch == "" {
		glog.V(100).Infof("The 'gitBranch' of the argocd application is empty")

		builder.errorMsg = "'gitBranch' parameter is empty"
	}

	if gitPath == "" {
		glog.V(100).Infof("The 'gitPath' of the argocd application is empty")

		builder.errorMsg = "'gitPath' parameter is empty"
	}

	glog.V(100).Infof(
		"Adding the following git details to the argocd application: %s in namespace: %s "+
			"RepoURL: %s,TargetRevision: %s, Path: %s", builder.Definition.Name, builder.Definition.Namespace,
		gitRepo, gitBranch, gitPath,
	)

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Source.RepoURL = gitRepo
	builder.Definition.Spec.Source.TargetRevision = gitBranch
	builder.Definition.Spec.Source.Path = gitPath

	return builder
}

// GetApplicationsGVR returns applications GroupVersionResource which could be used for Clean function.
func GetApplicationsGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group: APIGroup, Version: APIVersion, Resource: "applications",
	}
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *ApplicationBuilder) validate() (bool, error) {
	resourceCRD := "Application"

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

func (builder *ApplicationBuilder) convertToStructured(
	unsObject *unstructured.Unstructured) (*argocdtypes.Application, error) {
	application := &argocdtypes.Application{}

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unsObject.Object, application)
	if err != nil {
		glog.V(100).Infof(
			"Failed to convert from unstructured to Application object in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		return nil, err
	}

	return application, err
}
