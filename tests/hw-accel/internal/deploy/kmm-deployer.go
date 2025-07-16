package deploy

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
)

const (
	// KMMControllerDeployment name.
	KMMControllerDeployment = "kmm-operator-controller"
	// KMMWebhookDeployment name.
	KMMWebhookDeployment = "kmm-operator-webhook-server"
	// KMMLogLevel for logging.
	KMMLogLevel = 90
)

// KMMDeployer implements OperatorDeployer for KMM (operator only, no CRs).
type KMMDeployer struct {
	BaseOperatorDeployer
	CommonOps *CommonDeploymentOps
}

// NewKMMDeployer creates a new KMM deployer.
func NewKMMDeployer(config OperatorConfig) *KMMDeployer {
	base := BaseOperatorDeployer{Config: config}
	commonOps := NewCommonDeploymentOps(config)

	return &KMMDeployer{
		BaseOperatorDeployer: base,
		CommonOps:            commonOps,
	}
}

// deployOperatorGroupAllNamespaces creates operator group for AllNamespaces mode.
func (k *KMMDeployer) deployOperatorGroupAllNamespaces() error {
	operatorGroupBuilder := olm.NewOperatorGroupBuilder(
		k.Config.APIClient,
		k.Config.OperatorGroupName,
		k.Config.Namespace)

	// Configure for AllNamespaces mode by setting empty target namespaces.
	operatorGroupBuilder.Definition.Spec.TargetNamespaces = []string{}

	_, err := operatorGroupBuilder.Create()

	return err
}

// Deploy implements OperatorDeployer interface.
func (k *KMMDeployer) Deploy() error {
	glog.V(KMMLogLevel).Infof("Starting KMM operator deployment in namespace %s", k.Config.Namespace)

	// Create namespace
	if err := k.CommonOps.CreateNamespaceIfNotExist(); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Deploy operator group for AllNamespaces mode (KMM requirement).
	if err := k.deployOperatorGroupAllNamespaces(); err != nil {
		return fmt.Errorf("failed to deploy operator group: %w", err)
	}

	// Deploy subscription.
	if err := k.CommonOps.DeploySubscription(); err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	return nil
}

// IsReady implements OperatorDeployer interface.
func (k *KMMDeployer) IsReady(timeout time.Duration) (bool, error) {
	// Check controller deployment
	controllerReady, err := k.CommonOps.IsDeploymentReady(KMMControllerDeployment, timeout)

	if err != nil || !controllerReady {
		return false, err
	}

	// Check webhook deployment.
	webhookReady, err := k.CommonOps.IsDeploymentReady(KMMWebhookDeployment, timeout)

	if err != nil || !webhookReady {
		return false, err
	}

	return true, nil
}

// Undeploy implements OperatorDeployer interface.
func (k *KMMDeployer) Undeploy() error {
	glog.V(KMMLogLevel).Infof("Undeploying KMM operator from namespace %s", k.Config.Namespace)

	// Find and delete CSV
	csvName, err := k.CommonOps.FindCSV()
	if err != nil {
		glog.V(KMMLogLevel).Infof("Error finding CSV: %s", err.Error())

		return err
	}

	// Delete CSV.
	csvBuilder, err := olm.PullClusterServiceVersion(k.Config.APIClient, csvName, k.Config.Namespace)
	if err == nil && csvBuilder != nil {
		if err := k.CommonOps.DeleteResource(csvBuilder, 60*time.Second); err != nil {
			return fmt.Errorf("failed to delete CSV: %w", err)
		}
	}

	// Delete subscription.
	subBuilder, err := olm.PullSubscription(k.Config.APIClient, k.Config.SubscriptionName, k.Config.Namespace)
	if err == nil && subBuilder != nil {
		if err := k.CommonOps.DeleteResource(subBuilder, 60*time.Second); err != nil {
			return fmt.Errorf("failed to delete subscription: %w", err)
		}
	}

	// Delete operator group.
	ogBuilder, err := olm.PullOperatorGroup(k.Config.APIClient, k.Config.OperatorGroupName, k.Config.Namespace)
	if err == nil && ogBuilder != nil {
		if err := k.CommonOps.DeleteResource(ogBuilder, 60*time.Second); err != nil {
			return fmt.Errorf("failed to delete operator group: %w", err)
		}
	}

	return nil
}
