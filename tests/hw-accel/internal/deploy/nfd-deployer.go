package deploy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	nodefeature "github.com/openshift-kni/eco-goinfra/pkg/nfd"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/nfdparams"
)

const (
	// NFDControllerDeployment name.
	NFDControllerDeployment = "nfd-controller-manager"
	// NFDMasterDeployment name.
	NFDMasterDeployment = "nfd-master"
)

// NFDDeployer implements OperatorDeployer and CustomResourceDeployer for NFD.
type NFDDeployer struct {
	BaseOperatorDeployer
	CommonOps *CommonDeploymentOps
}

// NFDConfig holds NFD-specific configuration.
type NFDConfig struct {
	EnableTopology bool
	Image          string
}

// NewNFDDeployer creates a new NFD deployer.
func NewNFDDeployer(config OperatorConfig) *NFDDeployer {
	deployer := &NFDDeployer{
		BaseOperatorDeployer: BaseOperatorDeployer{Config: config},
		CommonOps:            NewCommonDeploymentOps(config),
	}

	return deployer
}

// Deploy implements OperatorDeployer interface.
func (n *NFDDeployer) Deploy() error {
	glog.V(nfdparams.LogLevel).Infof("Starting NFD operator deployment in namespace %s", n.Config.Namespace)

	// Create namespace.
	glog.V(nfdparams.LogLevel).Infof("Step 1: Creating namespace %s", n.Config.Namespace)

	if err := n.CommonOps.CreateNamespaceIfNotExist(); err != nil {
		glog.V(nfdparams.LogLevel).Infof("ERROR: Failed to create namespace %s: %v", n.Config.Namespace, err)

		return fmt.Errorf("failed to create namespace: %w", err)
	}

	glog.V(nfdparams.LogLevel).Infof("SUCCESS: Namespace %s created/verified", n.Config.Namespace)

	// Create operator group.
	glog.V(nfdparams.LogLevel).Infof(
		"Step nfdparams.LogLevel: Creating OperatorGroup %s in namespace %s",
		n.Config.OperatorGroupName,
		n.Config.Namespace)

	if err := n.CommonOps.DeployOperatorGroup(); err != nil {
		glog.V(nfdparams.LogLevel).Infof("ERROR: Failed to create OperatorGroup %s: %v", n.Config.OperatorGroupName, err)

		return fmt.Errorf("failed to create operator group: %w", err)
	}

	glog.V(nfdparams.LogLevel).Infof("SUCCESS: OperatorGroup %s created", n.Config.OperatorGroupName)

	// Verify operator group exists.
	ogBuilder, err := olm.PullOperatorGroup(n.Config.APIClient, n.Config.OperatorGroupName, n.Config.Namespace)

	if err != nil {
		glog.V(nfdparams.LogLevel).Infof("WARNING: Cannot verify OperatorGroup %s: %v", n.Config.OperatorGroupName, err)
	} else {
		glog.V(nfdparams.LogLevel).Infof("VERIFIED: OperatorGroup %s exists with TargetNamespaces: %v",
			ogBuilder.Object.Name, ogBuilder.Object.Spec.TargetNamespaces)
	}

	// Create subscription.
	glog.V(nfdparams.LogLevel).Infof(
		"Step 3: Creating Subscription %s in namespace %s",
		n.Config.SubscriptionName,
		n.Config.Namespace)
	glog.V(nfdparams.LogLevel).Infof(
		"Subscription details: Package=%s, CatalogSource=%s, CatalogSourceNamespace=%s, Channel=%s",
		n.Config.PackageName,
		n.Config.CatalogSource,
		n.Config.CatalogSourceNamespace,
		n.Config.Channel)

	if err := n.CommonOps.DeploySubscription(); err != nil {
		glog.V(nfdparams.LogLevel).Infof("ERROR: Failed to create Subscription %s: %v", n.Config.SubscriptionName, err)

		return fmt.Errorf("failed to create subscription: %w", err)
	}

	glog.V(nfdparams.LogLevel).Infof("SUCCESS: Subscription %s created", n.Config.SubscriptionName)

	// Verify subscription exists.
	subBuilder, err := olm.PullSubscription(n.Config.APIClient, n.Config.SubscriptionName, n.Config.Namespace)
	if err != nil {
		glog.V(nfdparams.LogLevel).Infof("WARNING: Cannot verify Subscription %s: %v", n.Config.SubscriptionName, err)
	} else {
		glog.V(nfdparams.LogLevel).Infof("VERIFIED: Subscription %s exists with State: %s, CurrentCSV: %s",
			subBuilder.Object.Name, subBuilder.Object.Status.State, subBuilder.Object.Status.CurrentCSV)
	}

	glog.
		V(nfdparams.LogLevel).
		Infof("NFD operator deployment completed. Check cluster for CSV creation and operator pods.")

	return nil
}

// IsReady implements OperatorDeployer interface.
func (n *NFDDeployer) IsReady(timeout time.Duration) (bool, error) {
	glog.V(nfdparams.LogLevel).Infof("Checking NFD operator readiness with deployment name: '%s'", NFDControllerDeployment)

	// Defensive validation - this should never happen but helps debug
	if NFDControllerDeployment == "" {
		return false, fmt.Errorf("NFDControllerDeployment constant is empty")
	}

	if n.CommonOps == nil {
		return false, fmt.Errorf("CommonOps is nil")
	}

	return n.CommonOps.IsDeploymentReady(NFDControllerDeployment, timeout)
}

// Undeploy implements OperatorDeployer interface.
func (n *NFDDeployer) Undeploy() error {
	glog.V(nfdparams.LogLevel).Infof("Undeploying NFD operator from namespace %s", n.Config.Namespace)

	// Find and delete CSV.
	csvName, err := n.CommonOps.FindCSV()
	if err != nil {
		glog.V(nfdparams.LogLevel).Infof("Error finding CSV: %s", err.Error())

		return err
	}

	// Delete CSV.
	csvBuilder, err := olm.PullClusterServiceVersion(n.Config.APIClient, csvName, n.Config.Namespace)
	if err == nil && csvBuilder != nil {
		if err := n.CommonOps.DeleteResource(csvBuilder, 60*time.Second); err != nil {
			return fmt.Errorf("failed to delete CSV: %w", err)
		}
	}

	// Delete subscription
	subBuilder, err := olm.PullSubscription(n.Config.APIClient, n.Config.SubscriptionName, n.Config.Namespace)
	if err == nil && subBuilder != nil {
		if err := n.CommonOps.DeleteResource(subBuilder, 60*time.Second); err != nil {
			return fmt.Errorf("failed to delete subscription: %w", err)
		}
	}

	// Delete operator group
	ogBuilder, err := olm.PullOperatorGroup(n.Config.APIClient, n.Config.OperatorGroupName, n.Config.Namespace)
	if err == nil && ogBuilder != nil {
		if err := n.CommonOps.DeleteResource(ogBuilder, 60*time.Second); err != nil {
			return fmt.Errorf("failed to delete operator group: %w", err)
		}
	}

	return nil
}

// DeployCustomResource implements CustomResourceDeployer interface.
func (n *NFDDeployer) DeployCustomResource(name string, config interface{}) error {
	nfdConfig, ok := config.(NFDConfig)
	if !ok {
		return fmt.Errorf("invalid config type for NFD, expected NFDConfig")
	}

	nfdBuilder, err := n.createNFDBuilder(nfdConfig)
	if err != nil {
		return fmt.Errorf("failed to create NFD builder: %w", err)
	}

	if name != "" {
		nfdBuilder.Definition.Name = name
	}

	glog.V(nfdparams.LogLevel).Infof("Deploying NFD CR: %s", nfdBuilder.Definition.Name)
	_, err = nfdBuilder.Create()

	if err != nil {
		return fmt.Errorf("failed to create NFD CR: %w", err)
	}

	return nil
}

// DeleteCustomResource implements CustomResourceDeployer interface.
func (n *NFDDeployer) DeleteCustomResource(name string) error {
	nfdBuilder, err := nodefeature.Pull(n.Config.APIClient, name, n.Config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to pull NFD CR: %w", err)
	}

	if nfdBuilder == nil {
		return fmt.Errorf("NFD CR %s not found", name)
	}

	// Remove finalizers to ensure clean deletion.
	nfdBuilder.Definition.Finalizers = []string{}
	_, err = nfdBuilder.Update(true)

	if err != nil {
		return fmt.Errorf("failed to update NFD CR finalizers: %w", err)
	}

	_, err = nfdBuilder.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete NFD CR: %w", err)
	}

	return nil
}

// IsCustomResourceReady implements CustomResourceDeployer interface.
func (n *NFDDeployer) IsCustomResourceReady(name string, timeout time.Duration) (bool, error) {
	// Implementation would check NFD CR status
	// For now, we'll just check if it exists
	nfdBuilder, err := nodefeature.Pull(n.Config.APIClient, name, n.Config.Namespace)
	if err != nil {
		return false, err
	}

	return nfdBuilder != nil, nil
}

// createNFDBuilder creates NFD builder from CSV examples.
func (n *NFDDeployer) createNFDBuilder(config NFDConfig) (*nodefeature.Builder, error) {
	clusters, err := olm.ListClusterServiceVersion(n.Config.APIClient, n.Config.Namespace)
	if err != nil {
		return nil, err
	}

	if len(clusters) == 0 {
		return nil, fmt.Errorf("no CSV found in %s namespace", n.Config.Namespace)
	}

	// Find NFD CSV.
	var nfdCSV *olm.ClusterServiceVersionBuilder

	for _, csv := range clusters {
		if strings.Contains(csv.Object.Name, "nfd") {
			nfdCSV = csv

			break
		}
	}

	if nfdCSV == nil {
		return nil, fmt.Errorf("NFD CSV not found")
	}

	almExamples, err := nfdCSV.GetAlmExamples()
	if err != nil {
		return nil, err
	}

	// Filter and edit ALM examples.
	almExamples, err = n.editAlmExample(almExamples)
	if err != nil {
		return nil, err
	}

	nfdBuilder := nodefeature.NewBuilderFromObjectString(n.Config.APIClient, almExamples)
	nfdBuilder.Definition.Spec.TopologyUpdater = config.EnableTopology

	if config.Image != "" {
		nfdBuilder.Definition.Spec.Operand.Image = config.Image
	}

	return nfdBuilder, nil
}

// editAlmExample filters ALM examples to include only NodeFeatureDiscovery.
func (n *NFDDeployer) editAlmExample(almExample string) (string, error) {
	var items []map[string]interface{}
	err := json.Unmarshal([]byte(almExample), &items)

	if err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	var filtered []map[string]interface{}

	for _, item := range items {
		if kind, ok := item["kind"]; ok && kind == "NodeFeatureDiscovery" {
			filtered = append(filtered, item)
		}
	}

	output, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal filtered JSON: %w", err)
	}

	return string(output), nil
}
