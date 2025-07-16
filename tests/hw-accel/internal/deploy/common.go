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

const (
	// LogLevel for logging.
	LogLevel = 90
)

// CommonDeploymentOps provides common deployment operations.
type CommonDeploymentOps struct {
	Config OperatorConfig
}

// NewCommonDeploymentOps creates a new common deployment operations instance.
func NewCommonDeploymentOps(config OperatorConfig) *CommonDeploymentOps {
	return &CommonDeploymentOps{Config: config}
}

// CreateNamespaceIfNotExist creates namespace if it doesn't exist.
func (c *CommonDeploymentOps) CreateNamespaceIfNotExist() error {
	nsbuilder := ns.NewBuilder(c.Config.APIClient, c.Config.Namespace)

	// Check if namespace already exists.
	if nsbuilder.Exists() {
		glog.V(LogLevel).Infof("Namespace %s already exists", c.Config.Namespace)

		return nil
	}

	glog.V(LogLevel).Infof("Creating namespace %s", c.Config.Namespace)

	_, err := nsbuilder.Create()

	if err != nil {
		glog.V(LogLevel).Infof("Failed to create namespace %s: %s", c.Config.Namespace, err.Error())

		return err
	}

	glog.V(LogLevel).Infof("Successfully created namespace %s", c.Config.Namespace)

	return nil
}

// DeployOperatorGroup creates operator group.
func (c *CommonDeploymentOps) DeployOperatorGroup() error {
	glog.V(LogLevel).Infof("Creating OperatorGroup %s in namespace %s", c.Config.OperatorGroupName, c.Config.Namespace)

	operatorGroupBuilder := olm.NewOperatorGroupBuilder(
		c.Config.APIClient,
		c.Config.OperatorGroupName,
		c.Config.Namespace)

	// Check if operator group already exists.
	if operatorGroupBuilder.Exists() {
		glog.V(LogLevel).Infof("OperatorGroup %s already exists", c.Config.OperatorGroupName)

		return nil
	}

	glog.V(LogLevel).Infof("OperatorGroup %s does not exist, creating it", c.Config.OperatorGroupName)

	_, err := operatorGroupBuilder.Create()

	if err != nil {
		glog.V(LogLevel).Infof("Failed to create OperatorGroup %s: %v", c.Config.OperatorGroupName, err)

		return err
	}

	glog.V(LogLevel).Infof("Successfully created OperatorGroup %s", c.Config.OperatorGroupName)

	return nil
}

// DeploySubscription creates subscription.
func (c *CommonDeploymentOps) DeploySubscription() error {
	glog.V(LogLevel).Infof("Creating Subscription %s in namespace %s", c.Config.SubscriptionName, c.Config.Namespace)
	glog.V(LogLevel).Infof("Subscription config: Package=%s, CatalogSource=%s, CatalogSourceNamespace=%s, Channel=%s",
		c.Config.PackageName, c.Config.CatalogSource, c.Config.CatalogSourceNamespace, c.Config.Channel)

	sub := olm.NewSubscriptionBuilder(
		c.Config.APIClient,
		c.Config.SubscriptionName,
		c.Config.Namespace,
		c.Config.CatalogSource,
		c.Config.CatalogSourceNamespace,
		c.Config.PackageName)
	sub.WithChannel(c.Config.Channel)

	// Check if subscription already exists.
	if sub.Exists() {
		glog.V(LogLevel).Infof("Subscription %s already exists", c.Config.SubscriptionName)

		return nil
	}

	glog.V(LogLevel).Infof("Subscription %s does not exist, creating it", c.Config.SubscriptionName)

	_, err := sub.Create()

	if err != nil {
		glog.V(LogLevel).Infof("Failed to create Subscription %s: %v", c.Config.SubscriptionName, err)

		return err
	}

	glog.V(LogLevel).Infof("Successfully created Subscription %s", c.Config.SubscriptionName)

	return nil
}

// IsDeploymentReady checks if a deployment is ready.
func (c *CommonDeploymentOps) IsDeploymentReady(deploymentName string, timeout time.Duration) (bool, error) {
	// Add defensive validation
	if deploymentName == "" {
		return false, fmt.Errorf("deployment name cannot be empty")
	}

	if c.Config.Namespace == "" {
		return false, fmt.Errorf("namespace cannot be empty")
	}

	if c.Config.APIClient == nil {
		return false, fmt.Errorf("API client cannot be nil")
	}

	glog.V(LogLevel).Infof("Checking deployment readiness: name=%s, namespace=%s", deploymentName, c.Config.Namespace)

	var deploymentBuilder *deploymentbuilder.Builder

	var err error

	timeoutErr := wait.PollUntilContextTimeout(
		context.TODO(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			deploymentBuilder, err = deploymentbuilder.Pull(c.Config.APIClient, deploymentName, c.Config.Namespace)
			if err != nil {
				glog.V(LogLevel).Infof(
					"Failed to pull deployment %s from namespace %s: %v",
					deploymentName,
					c.Config.Namespace, err)

				return false, nil // Continue polling
			}

			if deploymentBuilder == nil {
				glog.V(LogLevel).Infof("Deployment %s not found in namespace %s", deploymentName, c.Config.Namespace)

				return false, nil
			}

			if !deploymentBuilder.IsReady(timeout) {
				glog.V(LogLevel).Infof("Deployment %s in namespace %s not ready yet", deploymentName, c.Config.Namespace)
				err = fmt.Errorf("deployment %s isn't ready", deploymentName)

				return false, nil // Continue polling instead of failing immediately
			}

			glog.V(LogLevel).Infof("Deployment %s in namespace %s is ready", deploymentName, c.Config.Namespace)

			return true, nil
		})

	if timeoutErr != nil {
		if err != nil {
			return false, fmt.Errorf("deployment %s readiness check timed out after %v: %w", deploymentName, timeout, err)
		}

		return false, fmt.Errorf("deployment %s readiness check timed out after %v", deploymentName, timeout)
	}

	return true, nil
}

// DeleteResource deletes a resource and waits for completion.
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

// FindCSV finds the CSV in the namespace.
func (c *CommonDeploymentOps) FindCSV() (string, error) {
	clusterServices, err := olm.ListClusterServiceVersion(c.Config.APIClient, c.Config.Namespace)
	if err == nil && len(clusterServices) > 0 {
		return clusterServices[0].Definition.Name, nil
	}

	return "", err
}
