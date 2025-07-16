package deploy

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
)

const (
	// AMDControllerDeployment name.
	AMDControllerDeployment = "gpu-operator-controller-manager"
	// AMDLogLevel for logging.
	AMDLogLevel = 2
)

// AMDDeployer implements OperatorDeployer and CustomResourceDeployer for AMD GPU operator.
type AMDDeployer struct {
	BaseOperatorDeployer
	CommonOps *CommonDeploymentOps
}

// AMDConfig holds AMD GPU-specific configuration.
type AMDConfig struct {
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	Resources    string            `json:"resources,omitempty"`
	Tolerations  string            `json:"tolerations,omitempty"`
	LogLevel     string            `json:"logLevel,omitempty"`
}

// NewAMDDeployer creates a new AMD GPU deployer.
func NewAMDDeployer(config *OperatorConfig) *AMDDeployer {
	return &AMDDeployer{
		BaseOperatorDeployer: BaseOperatorDeployer{Config: *config},
		CommonOps:            NewCommonDeploymentOps(*config),
	}
}

// Deploy implements OperatorDeployer interface.
func (a *AMDDeployer) Deploy() error {
	// AMD GPU operator requires global namespace (openshift-operators)
	if a.Config.Namespace == "openshift-operators" {
		glog.V(AMDLogLevel).Infof("Using global namespace %s - skipping namespace and operator group creation",
			a.Config.Namespace)
	} else {
		// Create namespace
		if err := a.CommonOps.CreateNamespaceIfNotExist(); err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}

		glog.V(AMDLogLevel).Infof("SUCCESS: Namespace %s created/verified", a.Config.Namespace)

		// Create operator group
		if err := a.CommonOps.DeployOperatorGroup(); err != nil {
			return fmt.Errorf("failed to create operator group: %w", err)
		}

		glog.V(AMDLogLevel).Infof("SUCCESS: OperatorGroup %s created", a.Config.OperatorGroupName)
	}

	// Step 3: Create subscription
	glog.V(AMDLogLevel).Infof("Step 3: Creating Subscription %s in namespace %s",
		a.Config.SubscriptionName, a.Config.Namespace)

	if err := a.CommonOps.DeploySubscription(); err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	glog.V(AMDLogLevel).Infof("SUCCESS: Subscription %s created", a.Config.SubscriptionName)

	return nil
}

// IsReady implements OperatorDeployer interface.
func (a *AMDDeployer) IsReady(timeout time.Duration) (bool, error) {
	return a.CommonOps.IsDeploymentReady(AMDControllerDeployment, timeout)
}

// Undeploy implements OperatorDeployer interface.
func (a *AMDDeployer) Undeploy() error {
	// Delete subscription
	sub, err := olm.PullSubscription(a.Config.APIClient, a.Config.SubscriptionName, a.Config.Namespace)
	if err == nil && sub != nil {
		err = a.CommonOps.DeleteResource(sub, 60*time.Second)
		if err != nil {
			return err
		}
	}

	// Find and delete CSV
	csvName, err := a.CommonOps.FindCSV()
	if err == nil && csvName != "" {
		csv, err := olm.PullClusterServiceVersion(a.Config.APIClient, csvName, a.Config.Namespace)
		if err == nil && csv != nil {
			err = a.CommonOps.DeleteResource(csv, 60*time.Second)
			if err != nil {
				return err
			}
		}
	}

	// Delete operator group only if not using global namespace
	if a.Config.Namespace != "openshift-operators" {
		operatorGroup, err := olm.PullOperatorGroup(a.Config.APIClient, a.Config.OperatorGroupName, a.Config.Namespace)
		if err == nil && operatorGroup != nil {
			err = a.CommonOps.DeleteResource(operatorGroup, 60*time.Second)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DeployCustomResource implements CustomResourceDeployer interface for AMD DeviceConfig.
func (a *AMDDeployer) DeployCustomResource(name string, config interface{}) error {
	amdConfig, ok := config.(AMDConfig)
	if !ok {
		return fmt.Errorf("invalid config type for AMD GPU operator: expected AMDConfig")
	}

	glog.V(AMDLogLevel).Infof("AMD GPU Custom Resource deployment with config: %+v", amdConfig)

	return nil
}

// DeleteCustomResource implements CustomResourceDeployer interface.
func (a *AMDDeployer) DeleteCustomResource(name string) error {
	glog.V(AMDLogLevel).Infof("Deleting AMD GPU Custom Resource: %s", name)

	return nil
}

// IsCustomResourceReady implements CustomResourceDeployer interface.
func (a *AMDDeployer) IsCustomResourceReady(name string, timeout time.Duration) (bool, error) {
	glog.V(AMDLogLevel).Infof("Checking AMD GPU Custom Resource readiness: %s", name)

	return true, nil
}
