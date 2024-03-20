package service

import (
	"context"
	"fmt"

	"github.com/openshift-kni/eco-goinfra/pkg/msg"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Builder provides struct for service object containing connection to the cluster and the service definitions.
type Builder struct {
	// Service definition. Used to create a service object
	Definition *corev1.Service
	// Created service object
	Object *corev1.Service
	// Used in functions that define or mutate the service definition.
	// errorMsg is processed before the service object is created
	errorMsg  string
	apiClient *clients.Settings
}

// AdditionalOptions additional options for service object.
type AdditionalOptions func(builder *Builder) (*Builder, error)

// NewBuilder creates a new instance of Builder
// Default type of service is ClusterIP
// Use WithNodePort() for setting the NodePort type.
func NewBuilder(
	apiClient *clients.Settings,
	name string,
	nsname string,
	labels map[string]string,
	servicePort corev1.ServicePort) *Builder {
	glog.V(100).Infof(
		"Initializing new service structure with the following params: %s, %s", name, nsname)

	builder := Builder{
		apiClient: apiClient,
		Definition: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
			Spec: corev1.ServiceSpec{
				Selector: labels,
				Ports:    []corev1.ServicePort{servicePort},
			},
		},
	}

	if name == "" {
		glog.V(100).Infof("The name of the service is empty")

		builder.errorMsg = "Service 'name' cannot be empty"
	}

	if nsname == "" {
		glog.V(100).Infof("The namespace of the service is empty")

		builder.errorMsg = "Namespace 'nsname' cannot be empty"
	}

	return &builder
}

// WithNodePort redefines the service with NodePort service type.
func (builder *Builder) WithNodePort() *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	builder.Definition.Spec.Type = "NodePort"

	if len(builder.Definition.Spec.Ports) < 1 {
		builder.errorMsg = "service does not have the available ports"

		return builder
	}

	builder.Definition.Spec.Ports[0].NodePort = builder.Definition.Spec.Ports[0].Port

	return builder
}

// Pull loads an existing service into Builder struct.
func Pull(apiClient *clients.Settings, name, nsname string) (*Builder, error) {
	glog.V(100).Infof("Pulling existing service name: %s under namespace: %s", name, nsname)

	builder := Builder{
		apiClient: apiClient,
		Definition: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		builder.errorMsg = "service 'name' cannot be empty"
	}

	if nsname == "" {
		builder.errorMsg = "service 'namespace' cannot be empty"
	}

	if !builder.Exists() {
		return nil, fmt.Errorf("service object %s doesn't exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return &builder, nil
}

// Create the service in the cluster and store the created object in Object.
func (builder *Builder) Create() (*Builder, error) {
	if valid, err := builder.validate(); !valid {
		return builder, err
	}

	glog.V(100).Infof("Creating the service %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.Services(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metav1.CreateOptions{})
	}

	return builder, err
}

// Exists checks whether the given service exists.
func (builder *Builder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof(
		"Checking if service %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.Services(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metav1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// Delete a service.
func (builder *Builder) Delete() error {
	if valid, err := builder.validate(); !valid {
		return err
	}

	glog.V(100).Infof("Deleting the service %s from namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.Services(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metav1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// WithOptions creates service with generic mutation options.
func (builder *Builder) WithOptions(options ...AdditionalOptions) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Setting service additional options")

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

// WithExternalTrafficPolicy redefines the service with ServiceExternalTrafficPolicy type.
func (builder *Builder) WithExternalTrafficPolicy(policyType corev1.ServiceExternalTrafficPolicyType) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof(
		"Defining service's with ExternalTrafficPolicy: %v", policyType)

	if policyType == "" {
		glog.V(100).Infof(
			"Failed to set ExternalTrafficPolicy on service %s in namespace %s. "+
				"policyType can not be empty",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.errorMsg = "ExternalTrafficPolicy can not be empty"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.Type = "LoadBalancer"
	builder.Definition.Spec.ExternalTrafficPolicy = policyType

	return builder
}

// WithAnnotation redefines the service with Annotation type.
func (builder *Builder) WithAnnotation(annotation map[string]string) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Defining service's Annotation to %v", annotation)

	if annotation == nil {
		glog.V(100).Infof(
			"Failed to set Annotation on service %s in namespace %s. "+
				"Service Annotation can not be empty",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.errorMsg = "Annotation can not be empty map"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Annotations = annotation

	return builder
}

// WithIPFamily redefines the service with IPFamilies type.
func (builder *Builder) WithIPFamily(ipFamily []corev1.IPFamily, ipStackPolicy corev1.IPFamilyPolicyType) *Builder {
	if valid, _ := builder.validate(); !valid {
		return builder
	}

	glog.V(100).Infof("Defining service's IPFamily: %v and IPFamilyPolicy: %v", ipFamily, ipStackPolicy)

	if ipFamily == nil {
		glog.V(100).Infof("Failed to set empty ipFamily on service %s in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.errorMsg = "failed to set empty ipFamily"
	}

	if ipStackPolicy == "" {
		glog.V(100).Infof("Failed to set empty ipStackPolicy on service %s in namespace %s",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.errorMsg = "failed to set empty ipStackPolicy"
	}

	if builder.errorMsg != "" {
		return builder
	}

	builder.Definition.Spec.IPFamilies = ipFamily
	builder.Definition.Spec.IPFamilyPolicy = &ipStackPolicy

	return builder
}

// DefineServicePort helper for creating a Service with a ServicePort.
func DefineServicePort(port, targetPort int32, protocol corev1.Protocol) (*corev1.ServicePort, error) {
	glog.V(100).Infof(
		"Defining ServicePort with port %d and targetport %d", port, targetPort)

	if !isValidPort(port) {
		return nil, fmt.Errorf("invalid port number")
	}

	if !isValidPort(targetPort) {
		return nil, fmt.Errorf("invalid target port number")
	}

	return &corev1.ServicePort{
		Protocol: protocol,
		Port:     port,
		TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: targetPort,
		},
	}, nil
}

// GetServiceGVR returns service's GroupVersionResource which could be used for Clean function.
func GetServiceGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group: "", Version: "v1", Resource: "services",
	}
}

// isValidPort checks if a port is valid.
func isValidPort(port int32) bool {
	if (port > 0) || (port < 65535) {
		return true
	}

	return false
}

// validate will check that the builder and builder definition are properly initialized before
// accessing any member fields.
func (builder *Builder) validate() (bool, error) {
	resourceCRD := "Service"

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
