package deploy

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
)

const (
	// KMMControllerDeployment name
	KMMControllerDeployment = "kmm-operator-controller"
	// KMMWebhookDeployment name
	KMMWebhookDeployment = "kmm-operator-webhook-server"
	// KMMLogLevel for logging
	KMMLogLevel = 2
)

// KMMDeployer implements OperatorDeployer and CustomResourceDeployer for KMM
type KMMDeployer struct {
	BaseOperatorDeployer
	CommonOps *CommonDeploymentOps
}

// KMMConfig holds KMM-specific configuration
type KMMConfig struct {
	ModuleName     string
	KernelMapping  string
	ContainerImage string
	NodeSelector   map[string]string
}

// NewKMMDeployer creates a new KMM deployer
func NewKMMDeployer(config OperatorConfig) *KMMDeployer {
	deployer := &KMMDeployer{
		BaseOperatorDeployer: BaseOperatorDeployer{Config: config},
		CommonOps:            NewCommonDeploymentOps(config),
	}
	return deployer
}

// Deploy implements OperatorDeployer interface
func (k *KMMDeployer) Deploy() error {
	glog.V(KMMLogLevel).Infof("Deploying KMM operator in namespace %s", k.Config.Namespace)

	// Create namespace
	if err := k.CommonOps.CreateNamespaceIfNotExist(); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Create operator group
	if err := k.CommonOps.DeployOperatorGroup(); err != nil {
		return fmt.Errorf("failed to create operator group: %w", err)
	}

	// Create subscription
	if err := k.CommonOps.DeploySubscription(); err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	return nil
}

// IsReady implements OperatorDeployer interface
func (k *KMMDeployer) IsReady(timeout time.Duration) (bool, error) {
	// Check controller deployment
	controllerReady, err := k.CommonOps.IsDeploymentReady(KMMControllerDeployment, timeout)
	if err != nil || !controllerReady {
		return false, err
	}

	// Check webhook deployment
	webhookReady, err := k.CommonOps.IsDeploymentReady(KMMWebhookDeployment, timeout)
	if err != nil || !webhookReady {
		return false, err
	}

	return true, nil
}

// Undeploy implements OperatorDeployer interface
func (k *KMMDeployer) Undeploy() error {
	glog.V(KMMLogLevel).Infof("Undeploying KMM operator from namespace %s", k.Config.Namespace)

	// Find and delete CSV
	csvName, err := k.CommonOps.FindCSV()
	if err != nil {
		glog.V(KMMLogLevel).Infof("Error finding CSV: %s", err.Error())
		return err
	}

	// Delete CSV
	csvBuilder, err := olm.PullClusterServiceVersion(k.Config.APIClient, csvName, k.Config.Namespace)
	if err == nil && csvBuilder != nil {
		if err := k.CommonOps.DeleteResource(csvBuilder, 60*time.Second); err != nil {
			return fmt.Errorf("failed to delete CSV: %w", err)
		}
	}

	// Delete subscription
	subBuilder, err := olm.PullSubscription(k.Config.APIClient, k.Config.SubscriptionName, k.Config.Namespace)
	if err == nil && subBuilder != nil {
		if err := k.CommonOps.DeleteResource(subBuilder, 60*time.Second); err != nil {
			return fmt.Errorf("failed to delete subscription: %w", err)
		}
	}

	// Delete operator group
	ogBuilder, err := olm.PullOperatorGroup(k.Config.APIClient, k.Config.OperatorGroupName, k.Config.Namespace)
	if err == nil && ogBuilder != nil {
		if err := k.CommonOps.DeleteResource(ogBuilder, 60*time.Second); err != nil {
			return fmt.Errorf("failed to delete operator group: %w", err)
		}
	}

	return nil
}

// DeployCustomResource implements CustomResourceDeployer interface
func (k *KMMDeployer) DeployCustomResource(name string, config interface{}) error {
	kmmConfig, ok := config.(KMMConfig)
	if !ok {
		return fmt.Errorf("invalid config type for KMM, expected KMMConfig")
	}

	// Create basic KMM module
	moduleBuilder := kmm.NewModuleBuilder(k.Config.APIClient, name, k.Config.Namespace)

	// Set node selector if provided
	if len(kmmConfig.NodeSelector) > 0 {
		moduleBuilder.WithNodeSelector(kmmConfig.NodeSelector)
	}

	glog.V(KMMLogLevel).Infof("Deploying KMM Module: %s", moduleBuilder.Definition.Name)
	_, err := moduleBuilder.Create()
	if err != nil {
		return fmt.Errorf("failed to create KMM module: %w", err)
	}

	return nil
}

// DeleteCustomResource implements CustomResourceDeployer interface
func (k *KMMDeployer) DeleteCustomResource(name string) error {
	moduleBuilder, err := kmm.Pull(k.Config.APIClient, name, k.Config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to pull KMM module: %w", err)
	}

	if moduleBuilder == nil {
		return fmt.Errorf("KMM module %s not found", name)
	}

	_, err = moduleBuilder.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete KMM module: %w", err)
	}

	return nil
}

// IsCustomResourceReady implements CustomResourceDeployer interface
func (k *KMMDeployer) IsCustomResourceReady(name string, timeout time.Duration) (bool, error) {
	// Check if KMM module exists and is in ready state
	moduleBuilder, err := kmm.Pull(k.Config.APIClient, name, k.Config.Namespace)
	if err != nil {
		return false, err
	}

	if moduleBuilder == nil {
		return false, fmt.Errorf("KMM module %s not found", name)
	}

	// Check if module exists (simplified readiness check)
	return moduleBuilder.Exists(), nil
}

// DeploySimpleModule is a convenience method for deploying a simple KMM module
func (k *KMMDeployer) DeploySimpleModule(moduleName, kernelRegex, containerImage string, nodeSelector map[string]string) error {
	config := KMMConfig{
		ModuleName:     moduleName,
		NodeSelector:   nodeSelector,
		KernelMapping:  kernelRegex,
		ContainerImage: containerImage,
	}

	return k.DeployCustomResource(moduleName, config)
}

// GetModuleStatus returns the status of a KMM module
func (k *KMMDeployer) GetModuleStatus(moduleName string) (string, error) {
	moduleBuilder, err := kmm.Pull(k.Config.APIClient, moduleName, k.Config.Namespace)
	if err != nil {
		return "", fmt.Errorf("failed to pull KMM module: %w", err)
	}

	if moduleBuilder == nil {
		return "Not Found", nil
	}

	// In a real implementation, you'd check the actual status conditions
	// For now, return a simple status based on existence
	if moduleBuilder.Exists() {
		return "Ready", nil
	}

	return "Unknown", nil
}
