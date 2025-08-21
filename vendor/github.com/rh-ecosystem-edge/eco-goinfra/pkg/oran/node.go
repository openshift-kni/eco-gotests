package oran

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	hardwaremanagementv1alpha1 "github.com/openshift-kni/oran-o2ims/api/hardwaremanagement/v1alpha1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/msg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NodeBuilder provides a struct to inferface with Node resources on a specific cluster.
type NodeBuilder struct {
	// Definition of the Node used to create the resource.
	Definition *hardwaremanagementv1alpha1.Node
	// Object of the Node as it is on the cluster.
	Object *hardwaremanagementv1alpha1.Node
	// apiClient used to interact with the cluster.
	apiClient runtimeclient.Client
	// errorMsg used to store latest error message from functions that do not return errors.
	errorMsg string
}

// PullNode pulls an existing Node into a NodeBuilder struct.
func PullNode(apiClient *clients.Settings, name, nsname string) (*NodeBuilder, error) {
	glog.V(100).Infof("Pulling existing Node %s in namespace %s from cluster", name, nsname)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient of the Node is nil")

		return nil, fmt.Errorf("node 'apiClient' cannot be nil")
	}

	err := apiClient.AttachScheme(hardwaremanagementv1alpha1.AddToScheme)
	if err != nil {
		glog.V(100).Infof("Failed to add hardwaremanagement v1alpha1 scheme to client schemes: %v", err)

		return nil, err
	}

	builder := &NodeBuilder{
		apiClient: apiClient.Client,
		Definition: &hardwaremanagementv1alpha1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Info("The name of the Node is empty")

		return nil, fmt.Errorf("node 'name' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Info("The nsname of the Node is empty")

		return nil, fmt.Errorf("node 'nsname' cannot be empty")
	}

	if !builder.Exists() {
		glog.V(100).Info("The Node %s does not exist in namespace %s", name, nsname)

		return nil, fmt.Errorf("node object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return builder, nil
}

// Get returns the Node object if found.
func (builder *NodeBuilder) Get() (*hardwaremanagementv1alpha1.Node, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting Node object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	node := &hardwaremanagementv1alpha1.Node{}
	err := builder.apiClient.Get(context.TODO(), runtimeclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, node)

	if err != nil {
		glog.V(100).Infof("Failed to get Node object %s in namespace %s: %v",
			builder.Definition.Name, builder.Definition.Namespace, err)

		return nil, err
	}

	return node, nil
}

// Exists checks whether this Node exists on the cluster.
func (builder *NodeBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if Node %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate checks that the builder, definition, and apiClient are properly initialized and there is no errorMsg.
func (builder *NodeBuilder) validate() (bool, error) {
	resourceCRD := "node"

	if builder == nil {
		glog.V(100).Infof("The %s builder is uninitialized", resourceCRD)

		return false, fmt.Errorf("error: received nil %s builder", resourceCRD)
	}

	if builder.Definition == nil {
		glog.V(100).Infof("The %s is uninitialized", resourceCRD)

		return false, fmt.Errorf("%s", msg.UndefinedCrdObjectErrString(resourceCRD))
	}

	if builder.apiClient == nil {
		glog.V(100).Infof("The %s builder apiClient is nil", resourceCRD)

		return false, fmt.Errorf("%s builder cannot have nil apiClient", resourceCRD)
	}

	if builder.errorMsg != "" {
		glog.V(100).Infof("The %s builder has error message %s", resourceCRD, builder.errorMsg)

		return false, fmt.Errorf("%s", builder.errorMsg)
	}

	return true, nil
}
