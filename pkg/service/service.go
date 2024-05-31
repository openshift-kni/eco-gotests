package service

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const emptyDefinitionMsg = "no definition in builder"

// Builder provides struct for service object containing connection to the cluster and the service definitions.
type Builder struct {
	// Service definition. Used to create a service object
	Definition *v1.Service
	// Created service object
	Object *v1.Service
	// Used in functions that define or mutate the service definition.
	// errorMsg is processed before the service object is created
	errorMsg  string
	apiClient *clients.Settings
}

// NewBuilder creates a new instance of Builder
// Default type of service is ClusterIP
// Use WithNodePort() for setting the NodePort type.
func NewBuilder(
	apiClient *clients.Settings,
	name string,
	nsname string,
	labels map[string]string,
	servicePort v1.ServicePort) *Builder {
	glog.V(100).Infof(
		"Initializing new service structure with the following params: %s, %s", name, nsname)

	builder := Builder{
		apiClient: apiClient,
		Definition: &v1.Service{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
			Spec: v1.ServiceSpec{
				Selector: labels,
				Ports:    []v1.ServicePort{servicePort},
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
	if builder.Definition == nil {
		builder.errorMsg = emptyDefinitionMsg

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

// Create the service in the cluster and store the created object in Object.
func (builder *Builder) Create() (*Builder, error) {
	glog.V(100).Infof("Creating the service %s in namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if builder.errorMsg != "" {
		return nil, fmt.Errorf(builder.errorMsg)
	}

	var err error
	if !builder.Exists() {
		builder.Object, err = builder.apiClient.Services(builder.Definition.Namespace).Create(
			context.TODO(), builder.Definition, metaV1.CreateOptions{})
	}

	return builder, err
}

// Exists checks whether the given service exists.
func (builder *Builder) Exists() bool {
	glog.V(100).Infof(
		"Checking if service %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.apiClient.Services(builder.Definition.Namespace).Get(
		context.Background(), builder.Definition.Name, metaV1.GetOptions{})

	return err == nil || !k8serrors.IsNotFound(err)
}

// Delete a service.
func (builder *Builder) Delete() error {
	glog.V(100).Infof("Deleting the service %s from namespace %s", builder.Definition.Name, builder.Definition.Namespace)

	if !builder.Exists() {
		return nil
	}

	err := builder.apiClient.Services(builder.Definition.Namespace).Delete(
		context.TODO(), builder.Object.Name, metaV1.DeleteOptions{})

	if err != nil {
		return err
	}

	builder.Object = nil

	return err
}

// WithExternalTrafficPolicy redefines the service with ServiceExternalTrafficPolicy type.
func (builder *Builder) WithExternalTrafficPolicy(policyType v1.ServiceExternalTrafficPolicyType) *Builder {
	glog.V(100).Infof(
		"Defining service's with ExternalTrafficPolicy: %v", policyType)

	if builder.Definition == nil {
		glog.V(100).Infof(
			"Failed to set ExternalTrafficPolicy on service %s in namespace %s. "+
				"Service Definition can not be empty",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.errorMsg = emptyDefinitionMsg

		return builder
	}

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
	glog.V(100).Infof("Defining service's Annotation to %v", annotation)

	if builder.Definition == nil {
		glog.V(100).Infof(
			"Failed to set Annotation on service %s in namespace %s. "+
				"Service Definition can not be empty",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.errorMsg = emptyDefinitionMsg

		return builder
	}

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
func (builder *Builder) WithIPFamily(ipFamily []v1.IPFamily, ipStackPolicy v1.IPFamilyPolicyType) *Builder {
	glog.V(100).Infof("Defining service's IPFamily: %v and IPFamilyPolicy: %v", ipFamily, ipStackPolicy)

	if builder.Definition == nil {
		glog.V(100).Infof(
			"Failed to set IPFamily on service %s in namespace %s. "+
				"Service Definition can not be empty",
			builder.Definition.Name, builder.Definition.Namespace)

		builder.errorMsg = emptyDefinitionMsg

		return builder
	}

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
func DefineServicePort(port, targetPort int32, protocol v1.Protocol) (*v1.ServicePort, error) {
	glog.V(100).Infof(
		"Defining ServicePort with port %d and targetport %d", port, targetPort)

	if !isValidPort(port) {
		return nil, fmt.Errorf("invalid port number")
	}

	if !isValidPort(targetPort) {
		return nil, fmt.Errorf("invalid target port number")
	}

	return &v1.ServicePort{
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
