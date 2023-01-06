package service

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/clients"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

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
		builder.errorMsg = "no definition in builder"

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

// isValidPort checks if a port is valid.
func isValidPort(port int32) bool {
	if (port > 0) || (port < 65535) {
		return true
	}

	return false
}
