package deploy

import (
	"fmt"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
)

// DeployerType represents the type of deployer.
type DeployerType string

const (
	// NFDDeployerType for Node Feature Discovery.
	NFDDeployerType DeployerType = "nfd"
	// KMMDeployerType for Kernel Module Management.
	KMMDeployerType DeployerType = "kmm"
	// AMDDeployerType for AMD GPU operator.
	AMDDeployerType DeployerType = "amd"
)

// DeployerFactory provides methods to create deployers with standard configurations.
type DeployerFactory struct {
	APIClient *clients.Settings
}

// DeployerSet contains all available deployers.
type DeployerSet struct {
	NFD OperatorDeployer
	KMM OperatorDeployer
	AMD OperatorDeployer
}

// NewDeployerFactory creates a new deployer factory.
func NewDeployerFactory(apiClient *clients.Settings) *DeployerFactory {
	return &DeployerFactory{
		APIClient: apiClient,
	}
}

// CreateDeployer creates a specific deployer with default configuration.
func (f *DeployerFactory) CreateDeployer(deployerType DeployerType) (OperatorDeployer, error) {
	switch deployerType {
	case NFDDeployerType:
		return f.CreateNFDDeployer(nil), nil
	case KMMDeployerType:
		return f.CreateKMMDeployer(nil), nil
	case AMDDeployerType:
		return f.CreateAMDDeployer(nil), nil
	default:
		return nil, fmt.Errorf("unknown deployer type: %s", deployerType)
	}
}

// CreateAllDeployers creates all available deployers with default configurations.
func (f *DeployerFactory) CreateAllDeployers() *DeployerSet {
	return &DeployerSet{
		NFD: f.CreateNFDDeployer(nil),
		KMM: f.CreateKMMDeployer(nil),
		AMD: f.CreateAMDDeployer(nil),
	}
}

// CreateAllDeployersWithOverrides creates all deployers with custom configuration overrides.
func (f *DeployerFactory) CreateAllDeployersWithOverrides(
	nfdConfig, kmmConfig, amdConfig *OperatorConfig) *DeployerSet {
	return &DeployerSet{
		NFD: f.CreateNFDDeployer(nfdConfig),
		KMM: f.CreateKMMDeployer(kmmConfig),
		AMD: f.CreateAMDDeployer(amdConfig),
	}
}

// CreateNFDDeployer creates an NFD deployer with optional configuration override.
func (f *DeployerFactory) CreateNFDDeployer(override *OperatorConfig) *NFDDeployer {
	if f.APIClient == nil {
		panic("DeployerFactory APIClient cannot be nil")
	}

	config := f.getDefaultNFDConfig()
	if override != nil {
		config = f.mergeConfigs(config, *override)
	}

	// Validate final configuration
	if config.Namespace == "" {
		panic("NFD deployer namespace cannot be empty")
	}

	if config.APIClient == nil {
		panic("NFD deployer APIClient cannot be nil")
	}

	return NewNFDDeployer(config)
}

// CreateKMMDeployer creates a KMM deployer with optional configuration override.
func (f *DeployerFactory) CreateKMMDeployer(override *OperatorConfig) *KMMDeployer {
	config := f.getDefaultKMMConfig()
	if override != nil {
		config = f.mergeConfigs(config, *override)
	}

	return NewKMMDeployer(config)
}

// CreateAMDDeployer creates an AMD deployer with optional configuration override.
func (f *DeployerFactory) CreateAMDDeployer(override *OperatorConfig) *AMDDeployer {
	config := f.getDefaultAMDConfig()
	if override != nil {
		config = f.mergeConfigs(config, *override)
	}

	return NewAMDDeployer(&config)
}

// GetDeployerList returns a slice of all deployers for iteration.
func (ds *DeployerSet) GetDeployerList() []OperatorDeployer {
	return []OperatorDeployer{ds.NFD, ds.KMM, ds.AMD}
}

// GetDeployerMap returns a map of deployer type to deployer for lookup.
func (ds *DeployerSet) GetDeployerMap() map[DeployerType]OperatorDeployer {
	return map[DeployerType]OperatorDeployer{
		NFDDeployerType: ds.NFD,
		KMMDeployerType: ds.KMM,
		AMDDeployerType: ds.AMD,
	}
}

// Helper methods for default configurations.

func (f *DeployerFactory) getDefaultNFDConfig() OperatorConfig {
	return OperatorConfig{
		APIClient:              f.APIClient,
		Namespace:              "openshift-nfd",
		OperatorGroupName:      "nfd-operator-group",
		SubscriptionName:       "nfd-subscription",
		PackageName:            "nfd",
		CatalogSource:          "redhat-operators",
		CatalogSourceNamespace: "openshift-marketplace",
		Channel:                "stable",
		OperatorName:           "nfd-operator",
	}
}

func (f *DeployerFactory) getDefaultKMMConfig() OperatorConfig {
	return OperatorConfig{
		APIClient:              f.APIClient,
		Namespace:              "openshift-kmm",
		OperatorGroupName:      "kmm-operator-group",
		SubscriptionName:       "kmm-subscription",
		PackageName:            "kernel-module-management",
		CatalogSource:          "redhat-operators",
		CatalogSourceNamespace: "openshift-marketplace",
		Channel:                "stable",
		OperatorName:           "kmm-operator",
	}
}

func (f *DeployerFactory) getDefaultAMDConfig() OperatorConfig {
	return OperatorConfig{
		APIClient:              f.APIClient,
		Namespace:              "openshift-operators", // AMD GPU operator requires AllNamespaces install mode
		OperatorGroupName:      "global-operators",    // Use existing global operator group
		SubscriptionName:       "amd-gpu-subscription",
		PackageName:            "amd-gpu-operator",
		CatalogSource:          "certified-operators",
		CatalogSourceNamespace: "openshift-marketplace",
		Channel:                "alpha", // AMD GPU operator only has alpha channel available
		OperatorName:           "amd-gpu-operator",
	}
}

// mergeConfigs merges override config into base config (override takes precedence).
func (f *DeployerFactory) mergeConfigs(base, override OperatorConfig) OperatorConfig {
	result := base

	// Override non-empty values.
	if override.APIClient != nil {
		result.APIClient = override.APIClient
	}

	if override.Namespace != "" {
		result.Namespace = override.Namespace
	}

	if override.OperatorGroupName != "" {
		result.OperatorGroupName = override.OperatorGroupName
	}

	if override.SubscriptionName != "" {
		result.SubscriptionName = override.SubscriptionName
	}

	if override.PackageName != "" {
		result.PackageName = override.PackageName
	}

	if override.CatalogSource != "" {
		result.CatalogSource = override.CatalogSource
	}

	if override.CatalogSourceNamespace != "" {
		result.CatalogSourceNamespace = override.CatalogSourceNamespace
	}

	if override.Channel != "" {
		result.Channel = override.Channel
	}

	if override.OperatorName != "" {
		result.OperatorName = override.OperatorName
	}

	return result
}

// Convenience methods for common test scenarios.

// CreateTestDeployers creates deployers with test-friendly configurations.
func (f *DeployerFactory) CreateTestDeployers() *DeployerSet {
	// Use test-specific namespaces to avoid conflicts
	nfdOverride := &OperatorConfig{
		Namespace: "test-nfd",
	}
	kmmOverride := &OperatorConfig{
		Namespace: "test-kmm",
	}
	amdOverride := &OperatorConfig{
		Namespace: "test-amd-gpu",
	}

	return f.CreateAllDeployersWithOverrides(nfdOverride, kmmOverride, amdOverride)
}

// CreateProductionDeployers creates deployers with production configurations.
func (f *DeployerFactory) CreateProductionDeployers() *DeployerSet {
	// Use production-specific configurations if needed.
	return f.CreateAllDeployers()
}

// DeployAll deploys all operators in the deployer set.
func (ds *DeployerSet) DeployAll() error {
	deployers := ds.GetDeployerList()

	for _, deployer := range deployers {
		if err := deployer.Deploy(); err != nil {
			return fmt.Errorf("failed to deploy %s: %w", deployer.GetOperatorName(), err)
		}
	}

	return nil
}

// UndeployAll undeploys all operators in the deployer set.
func (ds *DeployerSet) UndeployAll() error {
	deployers := ds.GetDeployerList()

	var errors []error

	for i := len(deployers) - 1; i >= 0; i-- {
		if err := deployers[i].Undeploy(); err != nil {
			errors = append(errors, fmt.Errorf("failed to undeploy %s: %w", deployers[i].GetOperatorName(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("undeploy errors: %v", errors)
	}

	return nil
}
