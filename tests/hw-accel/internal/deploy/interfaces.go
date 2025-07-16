package deploy

import (
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
)

// OperatorDeployer defines the interface for operator deployment operations.
type OperatorDeployer interface {
	Deploy() error
	Undeploy() error
	IsReady(timeout time.Duration) (bool, error)
	GetNamespace() string
	GetOperatorName() string
}

// CustomResourceDeployer defines the interface for custom resource operations.
type CustomResourceDeployer interface {
	DeployCustomResource(name string, config interface{}) error
	DeleteCustomResource(name string) error
	IsCustomResourceReady(name string, timeout time.Duration) (bool, error)
}

// OperatorConfig holds common configuration for operator deployments.
type OperatorConfig struct {
	APIClient              *clients.Settings
	Namespace              string
	OperatorGroupName      string
	SubscriptionName       string
	PackageName            string
	CatalogSource          string
	CatalogSourceNamespace string
	Channel                string
	OperatorName           string
}

// BaseOperatorDeployer provides common functionality for operator deployments.
type BaseOperatorDeployer struct {
	Config OperatorConfig
}

// GetNamespace returns the namespace for the operator.
func (b *BaseOperatorDeployer) GetNamespace() string {
	return b.Config.Namespace
}

// GetOperatorName returns the operator name.
func (b *BaseOperatorDeployer) GetOperatorName() string {
	return b.Config.OperatorName
}

// NewOperatorConfig creates a new operator configuration.
func NewOperatorConfig(
	apiClient *clients.Settings,
	namespace string,
	operatorGroupName string,
	subscriptionName string,
	packageName string,
	catalogSource string,
	catalogSourceNamespace string,
	channel string,
	operatorName string) OperatorConfig {
	return OperatorConfig{
		APIClient:              apiClient,
		Namespace:              namespace,
		OperatorGroupName:      operatorGroupName,
		SubscriptionName:       subscriptionName,
		PackageName:            packageName,
		CatalogSource:          catalogSource,
		CatalogSourceNamespace: catalogSourceNamespace,
		Channel:                channel,
		OperatorName:           operatorName,
	}
}
