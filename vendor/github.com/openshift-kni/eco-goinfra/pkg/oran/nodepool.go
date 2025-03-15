package oran

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/msg"
	hardwaremanagementv1alpha1 "github.com/openshift-kni/oran-o2ims/api/hardwaremanagement/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NodePoolBuilder provides a struct to inferface with NodePool resources on a specific cluster.
type NodePoolBuilder struct {
	// Definition of the NodePool used to create the resource.
	Definition *hardwaremanagementv1alpha1.NodePool
	// Object of the NodePool as it is on the cluster.
	Object *hardwaremanagementv1alpha1.NodePool
	// apiClient used to interact with the cluster.
	apiClient runtimeclient.Client
	// errorMsg used to store latest error message from functions that do not return errors.
	errorMsg string
}

// PullNodePool pulls an existing NodePool into a NodePoolBuilder struct.
func PullNodePool(apiClient *clients.Settings, name, nsname string) (*NodePoolBuilder, error) {
	glog.V(100).Infof("Pulling existing NodePool %s in namespace %s from cluster", name, nsname)

	if apiClient == nil {
		glog.V(100).Infof("The apiClient of the NodePool is nil")

		return nil, fmt.Errorf("nodePool 'apiClient' cannot be nil")
	}

	err := apiClient.AttachScheme(hardwaremanagementv1alpha1.AddToScheme)
	if err != nil {
		glog.V(100).Infof("Failed to add hardwaremanagement v1alpha1 scheme to client schemes: %v", err)

		return nil, err
	}

	builder := &NodePoolBuilder{
		apiClient: apiClient.Client,
		Definition: &hardwaremanagementv1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: nsname,
			},
		},
	}

	if name == "" {
		glog.V(100).Info("The name of the NodePool is empty")

		return nil, fmt.Errorf("nodePool 'name' cannot be empty")
	}

	if nsname == "" {
		glog.V(100).Info("The nsname of the NodePool is empty")

		return nil, fmt.Errorf("nodePool 'nsname' cannot be empty")
	}

	if !builder.Exists() {
		glog.V(100).Info("The NodePool %s does not exist in namespace %s", name, nsname)

		return nil, fmt.Errorf("nodePool object %s does not exist in namespace %s", name, nsname)
	}

	builder.Definition = builder.Object

	return builder, nil
}

// Get returns the NodePool object if found.
func (builder *NodePoolBuilder) Get() (*hardwaremanagementv1alpha1.NodePool, error) {
	if valid, err := builder.validate(); !valid {
		return nil, err
	}

	glog.V(100).Infof("Getting NodePool object %s in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	nodePool := &hardwaremanagementv1alpha1.NodePool{}
	err := builder.apiClient.Get(context.TODO(), runtimeclient.ObjectKey{
		Name:      builder.Definition.Name,
		Namespace: builder.Definition.Namespace,
	}, nodePool)

	if err != nil {
		glog.V(100).Infof("Failed to get NodePool object %s in namespace %s: %v",
			builder.Definition.Name, builder.Definition.Namespace, err)

		return nil, err
	}

	return nodePool, nil
}

// Exists checks whether this NodePool exists on the cluster.
func (builder *NodePoolBuilder) Exists() bool {
	if valid, _ := builder.validate(); !valid {
		return false
	}

	glog.V(100).Infof("Checking if NodePool %s exists in namespace %s",
		builder.Definition.Name, builder.Definition.Namespace)

	var err error
	builder.Object, err = builder.Get()

	return err == nil || !k8serrors.IsNotFound(err)
}

// validate checks that the builder, definition, and apiClient are properly initialized and there is no errorMsg.
func (builder *NodePoolBuilder) validate() (bool, error) {
	resourceCRD := "nodePool"

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
