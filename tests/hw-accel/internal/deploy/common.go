package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	deploymentbuilder "github.com/openshift-kni/eco-goinfra/pkg/deployment"
	ns "github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"k8s.io/apimachinery/pkg/util/wait"
)

// CommonDeploymentOps provides common deployment operations
type CommonDeploymentOps struct {
	Config OperatorConfig
}

// NewCommonDeploymentOps creates a new common deployment operations instance
func NewCommonDeploymentOps(config OperatorConfig) *CommonDeploymentOps {
	return &CommonDeploymentOps{Config: config}
}

// CreateNamespaceIfNotExist creates namespace if it doesn't exist
func (c *CommonDeploymentOps) CreateNamespaceIfNotExist() error {
	nsbuilder := ns.NewBuilder(c.Config.APIClient, c.Config.Namespace)
	_, err := nsbuilder.Create()
	if err != nil {
		glog.V(2).Infof("Namespace might already exist or failed to create: %s", err.Error())
	}
	return nil
}

// DeployOperatorGroup creates operator group
func (c *CommonDeploymentOps) DeployOperatorGroup() error {
	operatorGroupBuilder := olm.NewOperatorGroupBuilder(
		c.Config.APIClient,
		c.Config.OperatorGroupName,
		c.Config.Namespace)

	_, err := operatorGroupBuilder.Create()
	return err
}

// DeploySubscription creates subscription
func (c *CommonDeploymentOps) DeploySubscription() error {
	sub := olm.NewSubscriptionBuilder(
		c.Config.APIClient,
		c.Config.SubscriptionName,
		c.Config.Namespace,
		c.Config.CatalogSource,
		c.Config.CatalogSourceNamespace,
		c.Config.PackageName)
	sub.WithChannel(c.Config.Channel)

	_, err := sub.Create()
	return err
}

// IsDeploymentReady checks if a deployment is ready
func (c *CommonDeploymentOps) IsDeploymentReady(deploymentName string, timeout time.Duration) (bool, error) {
	var deploymentBuilder *deploymentbuilder.Builder
	var err error

	timeoutErr := wait.PollUntilContextTimeout(
		context.TODO(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			deploymentBuilder, err = deploymentbuilder.Pull(c.Config.APIClient, deploymentName, c.Config.Namespace)
			if deploymentBuilder == nil {
				return false, nil
			}

			if !deploymentBuilder.IsReady(timeout) {
				err = fmt.Errorf("deployment %s isn't ready", deploymentName)
				return false, err
			}

			return true, nil
		})

	if timeoutErr != nil {
		return false, err
	}

	return true, nil
}

// DeleteResource deletes a resource and waits for completion
func (c *CommonDeploymentOps) DeleteResource(builder builder, timeout time.Duration) error {
	if err := builder.Delete(); err != nil {
		return err
	}

	return wait.PollUntilContextTimeout(
		context.TODO(), time.Second, timeout*5, true, func(ctx context.Context) (bool, error) {
			isFound := builder.Exists()
			if isFound {
				return false, nil
			}
			return true, nil
		})
}

// FindCSV finds the CSV in the namespace
func (c *CommonDeploymentOps) FindCSV() (string, error) {
	clusterServices, err := olm.ListClusterServiceVersion(c.Config.APIClient, c.Config.Namespace)
	if err == nil && len(clusterServices) > 0 {
		return clusterServices[0].Definition.Name, nil
	}
	return "", err
}
