package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	deploymentbuilder "github.com/openshift-kni/eco-goinfra/pkg/deployment"
	ns "github.com/openshift-kni/eco-goinfra/pkg/namespace"
	nodefeature "github.com/openshift-kni/eco-goinfra/pkg/nfd"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/nfdparams"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// NfdController nfd deployment name.
	NfdController = "nfd-controller-manager"
	// NfdMaster nfd cr deployment name.
	NfdMaster = "nfd-master"

	logLevel = nfdparams.LogLevel
)

const (
	// OperatorGroup enum value type.
	OperatorGroup builderType = iota
	// NodeFeatureDiscovery enum value type.
	NodeFeatureDiscovery
	// Subscription enum value type.
	Subscription
	// ClusterVersion enum value type.
	ClusterVersion
	// NameSpace enum value type.
	NameSpace
)

type builderType int

type builder interface {
	Delete() error
	Exists() bool
}

type nfdAdapter struct {
	nodeFeatureBuilder *nodefeature.Builder
}

func deleteAndWait(builder builder, timeout time.Duration) error {
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

// NfdAPIResource object that represents NodeFeatureDiscovery resource with API client.
type NfdAPIResource struct {
	APIClients             *clients.Settings
	Namespace              string
	OperatorGroupName      string
	SubName                string
	CatalogSource          string
	CatalogSourceNamespace string
	PackageName            string
	Channel                string
}

// NewNfdAPIResource create NodeFeatureDiscovery api client that contains all related field for the resource.
func NewNfdAPIResource(
	apiClient *clients.Settings,
	namespace,
	operatorGroupName,
	subName,
	catalogSource,
	catalogSourceNamespace,
	packageName,
	channel string) *NfdAPIResource {
	return &NfdAPIResource{
		APIClients:             apiClient,
		Namespace:              namespace,
		OperatorGroupName:      operatorGroupName,
		SubName:                subName,
		CatalogSource:          catalogSource,
		CatalogSourceNamespace: catalogSourceNamespace,
		PackageName:            packageName,
		Channel:                channel,
	}
}

// DeployNfd deploy NodeFeatureDiscovery operator and cr return error if it failed.
func (n *NfdAPIResource) DeployNfd(waitTime int, addTopology bool, nfdInstanceImage string) error {
	glog.V(logLevel).Infof(
		"Deploying node feature discovery")

	err := n.deploy()
	if err != nil {
		glog.V(logLevel).Infof(
			"Error in Deploying NodeFeatureDiscovery : %s", err.Error())

		return err
	}

	deploymentReady, err := n.IsDeploymentReady(time.Minute*time.Duration(waitTime), NfdController)

	if err != nil {
		glog.V(logLevel).Infof(
			"Error %s not found\n cause: %s", NfdController, err.Error())

		return err
	}

	if !deploymentReady {
		return fmt.Errorf("nfd deployment didn't become ready within the specified timeout")
	}

	err = deployNfdCR(n.Namespace, addTopology, nfdInstanceImage)
	if err != nil {
		glog.V(logLevel).Infof(
			"Error in deploying NodeFeatureDiscovery CR cause: %s", err.Error())

		return err
	}

	return nil
}

// UndeployNfd remove nfd completely instance name.
func (n *NfdAPIResource) UndeployNfd(nodeFeatureName string) error {
	csvName, err := findCSV(n.Namespace)
	if err != nil {
		glog.V(logLevel).Infof("Error in find CSV cause: %s", err.Error())

		return err
	}

	err = n.removeResource(nodeFeatureName, NodeFeatureDiscovery)
	if err != nil {
		glog.V(logLevel).Infof("Error removing resource %s cause: %s",
			nodeFeatureName, err.Error())

		return err
	}

	err = n.removeResource(csvName, ClusterVersion)
	if err != nil {
		glog.V(logLevel).Infof("Error removing resource %s cause: %s",
			csvName, err.Error())

		return err
	}

	err = n.removeResource(n.SubName, Subscription)
	if err != nil {
		glog.V(logLevel).Infof("Error removing resource %s cause: %s",
			n.SubName, err.Error())

		return err
	}

	err = n.removeResource(n.OperatorGroupName, OperatorGroup)
	if err != nil {
		glog.V(logLevel).Infof("Error removing resource %s cause: %s",
			n.OperatorGroupName, err.Error())

		return err
	}

	err = n.removeResource("", NameSpace)
	if err != nil {
		glog.V(logLevel).Infof("Error removing resource %s cause: %s",
			n.Namespace, err.Error())

		return err
	}

	return nil
}

func (n nfdAdapter) Delete() error {
	_, err := n.nodeFeatureBuilder.Delete()

	return err
}

func (n nfdAdapter) Exists() bool {
	return n.nodeFeatureBuilder.Exists()
}

// IsDeploymentReady check and wait for nfd deployment status.
func (n *NfdAPIResource) IsDeploymentReady(waitTime time.Duration,
	deployment string) (bool, error) {
	deploymentBuilder, err := deploymentbuilder.Pull(n.APIClients, deployment, n.Namespace)

	timeOutError := wait.PollUntilContextTimeout(
		context.TODO(), time.Second, waitTime, true, func(ctx context.Context) (bool, error) {
			deploymentBuilder, err = deploymentbuilder.Pull(n.APIClients, deployment, n.Namespace)
			if deploymentBuilder == nil {
				return false, nil
			}

			if !deploymentBuilder.IsReady(waitTime) {
				err = fmt.Errorf("deployment %s isn't ready", deployment)

				return false, err
			}

			return true, nil
		})

	if timeOutError != nil {
		return false, err
	}

	return true, nil
}

func (n *NfdAPIResource) deploy() error {
	n.createNameSpaceIfNotExist()

	operatorGroupbuilder := olm.NewOperatorGroupBuilder(n.APIClients,
		n.OperatorGroupName, n.Namespace)

	sub := olm.NewSubscriptionBuilder(n.APIClients, n.SubName,
		n.Namespace, n.CatalogSource, n.CatalogSourceNamespace, n.PackageName)
	sub.WithChannel(n.Channel)

	_, err := operatorGroupbuilder.Create()
	if err != nil {
		return err
	}

	_, err = sub.Create()
	if err != nil {
		return err
	}

	return nil
}

func newNfdBuilder(namespace string, enableTopology bool, image string) (*nodefeature.Builder, error) {
	clusters, err := olm.ListClusterServiceVersion(APIClient, namespace)

	if err != nil {
		return nil, err
	}

	if len(clusters) == 0 {
		return nil, fmt.Errorf("no csv in %s namespace", namespace)
	}

	nfdcsv, err := olm.PullClusterServiceVersion(APIClient, clusters[0].Object.Name, namespace)

	if err != nil {
		return nil, err
	}

	almExamples, err := nfdcsv.GetAlmExamples()
	if err != nil {
		return nil, err
	}

	nfdBuilder := nodefeature.NewBuilderFromObjectString(APIClient, almExamples)
	nfdBuilder.Definition.Spec.TopologyUpdater = enableTopology

	if image != "" {
		nfdBuilder.Definition.Spec.Operand.Image = image
	}

	return nfdBuilder, nil
}

func deployNfdCR(namespace string, enableTopology bool, image string) error {
	nfdBuilder, err := newNfdBuilder(namespace, enableTopology, image)
	if err != nil {
		return err
	}

	_, err = nfdBuilder.Create()
	if err != nil {
		return err
	}

	return nil
}

// DeleteNFDCR removes node feature discovery worker.
func (n *NfdAPIResource) DeleteNFDCR(nodeFeatureName string) error {
	err := n.removeResource(nodeFeatureName, NodeFeatureDiscovery)
	if err != nil {
		glog.V(logLevel).Infof("Error removing resource %s cause: %s",
			nodeFeatureName, err.Error())

		return err
	}

	return nil
}

// DeployNfdWithCustomConfig deploys nfd worker with custom config.
func DeployNfdWithCustomConfig(namespace string, enableTopology bool, config string, image string) error {
	nfdBuilder, err := newNfdBuilder(namespace, enableTopology, image)
	if err != nil {
		return err
	}

	nfdBuilder.Definition.Spec.WorkerConfig.ConfigData = config

	_, err = nfdBuilder.Create()
	if err != nil {
		return err
	}

	return nil
}

func (n *NfdAPIResource) createNameSpaceIfNotExist() {
	nsbuilder := ns.NewBuilder(n.APIClients, n.Namespace)

	if _, err := nsbuilder.Create(); err != nil {
		glog.V(logLevel).Infof("Error in creating namespace cause: %s", err.Error())
	}
}

func findCSV(namespace string) (string, error) {
	clusterServices, err := olm.ListClusterServiceVersion(APIClient, namespace)

	if err == nil && len(clusterServices) > 0 {
		return clusterServices[0].Definition.Name, nil
	}

	return "", err
}

func (n *NfdAPIResource) removeResource(resourceName string,
	builderType builderType) error {
	var err error

	var builder builder

	var nfdbuilder *nodefeature.Builder

	switch builderType {
	case NameSpace:
		builder, err = ns.Pull(n.APIClients, n.Namespace)
	case OperatorGroup:
		builder, err = olm.PullOperatorGroup(n.APIClients,
			resourceName, n.Namespace)
	case NodeFeatureDiscovery:
		nfdbuilder, err = nodefeature.Pull(n.APIClients,
			resourceName, n.Namespace)
		if err != nil {
			return err
		}

		if nfdbuilder == nil {
			return fmt.Errorf("resource %v not found", resourceName)
		}

		nfdbuilder.Definition.Finalizers = []string{}
		_, updateErr := nfdbuilder.Update(true)

		if updateErr != nil {
			return err
		}

		nfdAdapt := nfdAdapter{nfdbuilder}

		builder = nfdAdapt
	case Subscription:
		builder, err = olm.PullSubscription(n.APIClients,
			resourceName, n.Namespace)

	case ClusterVersion:
		builder, err = olm.PullClusterServiceVersion(n.APIClients,
			resourceName, n.Namespace)
	}

	if err != nil {
		return err
	}

	if builder == nil {
		return fmt.Errorf("didn't found node feature %s instance in %s",
			resourceName, n.Namespace)
	}

	err = deleteAndWait(builder, 60*time.Second)

	if err != nil {
		return err
	}

	return nil
}
