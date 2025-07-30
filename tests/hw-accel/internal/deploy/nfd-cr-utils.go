package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	nodefeature "github.com/openshift-kni/eco-goinfra/pkg/nfd"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/nfd/nfdparams"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// NFDCRConfig holds configuration for NFD custom resource.
type NFDCRConfig struct {
	EnableTopology bool   `json:"enableTopology,omitempty"`
	Image          string `json:"image,omitempty"`
	WorkerConfig   string `json:"workerConfig,omitempty"`
}

// NFDCRUtils provides utilities for managing NFD custom resources.
type NFDCRUtils struct {
	APIClient *clients.Settings
	Namespace string
	LogLevel  glog.Level
	CrName    string
}

// NewNFDCRUtils creates a new NFD CR utilities instance.
func NewNFDCRUtils(apiClient *clients.Settings, namespace string, name string) *NFDCRUtils {
	return &NFDCRUtils{
		APIClient: apiClient,
		Namespace: namespace,
		LogLevel:  glog.Level(nfdparams.LogLevel),
		CrName:    name,
	}
}

// DeployNFDCR deploys a NodeFeatureDiscovery custom resource.
func (nfd *NFDCRUtils) DeployNFDCR(config NFDCRConfig) error {
	glog.V(nfd.LogLevel).Infof("Deploying NFD CR '%s' in namespace '%s'", nfd.CrName, nfd.Namespace)

	nfdBuilder, err := nfd.createNFDBuilder(config)
	if err != nil {
		return fmt.Errorf("failed to create NFD builder: %w", err)
	}

	if nfd.CrName != "" {
		nfdBuilder.Definition.Name = nfd.CrName
	}

	glog.V(nfd.LogLevel).Infof("Creating NFD CR: %v", nfdBuilder.Definition)
	_, err = nfdBuilder.Create()

	if err != nil {
		return fmt.Errorf("failed to create NFD CR: %w", err)
	}

	glog.V(nfd.LogLevel).Infof("SUCCESS: NFD CR '%s' deployed", nfdBuilder.Definition.Name)

	return nil
}

// DeleteNFDCR deletes a NodeFeatureDiscovery custom resource.
func (nfd *NFDCRUtils) DeleteNFDCR() error {
	glog.V(nfd.LogLevel).Infof("Deleting NFD CR '%s' from namespace '%s'", nfd.CrName, nfd.Namespace)

	nfdBuilder, err := nodefeature.Pull(nfd.APIClient, nfd.CrName, nfd.Namespace)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") ||
			strings.Contains(strings.ToLower(err.Error()), "does not exist") {
			glog.V(nfd.LogLevel).Infof("NFD CR '%s' not found - already deleted or never existed", nfd.CrName)

			return nil
		}

		return fmt.Errorf("failed to pull NFD CR: %w", err)
	}

	if nfdBuilder == nil {
		glog.V(nfd.LogLevel).Infof("NFD CR '%s' not found", nfd.CrName)

		return nil
	}

	nfdBuilder.Definition.Finalizers = []string{}

	_, err = nfdBuilder.Update(true)
	if err != nil {
		glog.V(nfd.LogLevel).Infof("Warning: failed to update NFD CR finalizers: %v", err)
	}

	_, err = nfdBuilder.Delete()
	if err != nil {
		return fmt.Errorf("failed to delete NFD CR: %w", err)
	}

	glog.V(nfd.LogLevel).Infof("SUCCESS: NFD CR '%s' deleted", nfd.CrName)

	return nil
}

// PrintCr prints the NFD CR configuration.
func (nfd *NFDCRUtils) PrintCr() error {
	nfdBuilder, err := nodefeature.Pull(nfd.APIClient, nfd.CrName, nfd.Namespace)

	if err != nil {
		glog.V(nfd.LogLevel).Infof("Failed to pull NFD CR: %v", err)

		return fmt.Errorf("failed to pull NFD CR: %w", err)
	}

	if nfdBuilder != nil {
		yml, err := yaml.Marshal(nfdBuilder.Definition)
		if err != nil {
			return fmt.Errorf("failed to marshal NFD CR: %w", err)
		}

		glog.Infof("NFD CR '%s' ", string(yml))

		return nil
	}

	return fmt.Errorf("NFD CR '%s' not found", nfd.CrName)
}

// IsNFDCRReady checks if a NodeFeatureDiscovery custom resource is ready.
func (nfd *NFDCRUtils) IsNFDCRReady(timeout time.Duration) (bool, error) {
	glog.V(nfd.LogLevel).Infof("Checking NFD CR readiness: %s", nfd.CrName)

	nfdBuilder, err := nodefeature.Pull(nfd.APIClient, nfd.CrName, nfd.Namespace)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") ||
			strings.Contains(strings.ToLower(err.Error()), "does not exist") {
			glog.V(nfd.LogLevel).Infof("NFD CR '%s' not found yet - not ready", nfd.CrName)

			return false, nil
		}

		return false, fmt.Errorf("failed to pull NFD CR: %w", err)
	}

	if nfdBuilder != nil {
		glog.V(nfd.LogLevel).Infof("NFD CR '%s' found - checking pods", nfdBuilder.Definition.Name)

		return ForPodsRunning(nfd.APIClient, timeout, nfd.Namespace)
	}

	return false, fmt.Errorf("NFD CR '%s' not found", nfd.CrName)
}

// ForPodsRunning checks that all pods in namespace are in running state.
func ForPodsRunning(apiClient *clients.Settings, timeout time.Duration, nsname string) (bool, error) {
	err := wait.PollUntilContextTimeout(
		context.TODO(), 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			pods, err := pod.List(apiClient, nsname, metav1.ListOptions{})
			if err != nil {
				return false, err
			}

			for _, podObj := range pods {
				if podObj.Object.Status.Phase != corev1.PodRunning {
					glog.V(nfdparams.LogLevel).Infof("Pod %s is in %s state", podObj.Object.Name, podObj.Object.Status.Phase)

					return false, nil
				}
			}

			glog.V(nfdparams.LogLevel).Info("All pods are in running status")

			return true, nil
		})

	if err != nil {
		return false, err
	}

	return true, nil
}

// DeployNFDCRWithWorkerConfig deploys NFD CR with custom worker configuration.
func (nfd *NFDCRUtils) DeployNFDCRWithWorkerConfig(config NFDCRConfig, workerConfig string) error {
	glog.V(nfd.LogLevel).Infof("Deploying NFD CR '%s' with custom worker config", nfd.CrName)

	nfdBuilder, err := nfd.createNFDBuilder(config)
	if err != nil {
		return fmt.Errorf("failed to create NFD builder: %w", err)
	}

	if nfd.CrName != "" {
		nfdBuilder.Definition.Name = nfd.CrName
	}

	if workerConfig != "" {
		nfdBuilder.Definition.Spec.WorkerConfig.ConfigData = workerConfig

		glog.V(nfd.LogLevel).Infof("Applied custom worker config to NFD CR")
	}

	_, err = nfdBuilder.Create()
	if err != nil {
		return fmt.Errorf("failed to create NFD CR with worker config: %w", err)
	}

	glog.V(nfd.LogLevel).Infof("SUCCESS: NFD CR '%v' with worker config deployed", nfdBuilder.Definition)

	return nil
}

// createNFDBuilder creates NFD builder from CSV examples.
func (nfd *NFDCRUtils) createNFDBuilder(config NFDCRConfig) (*nodefeature.Builder, error) {
	clusters, err := olm.ListClusterServiceVersion(nfd.APIClient, nfd.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list CSVs: %w", err)
	}

	if len(clusters) == 0 {
		return nil, fmt.Errorf("no CSV found in %s namespace", nfd.Namespace)
	}

	var nfdCSV *olm.ClusterServiceVersionBuilder

	for _, csv := range clusters {
		if strings.Contains(strings.ToLower(csv.Object.Name), "nfd") {
			nfdCSV = csv

			break
		}
	}

	if nfdCSV == nil {
		return nil, fmt.Errorf("NFD CSV not found in namespace %s", nfd.Namespace)
	}

	glog.V(nfd.LogLevel).Infof("Using NFD CSV: %s", nfdCSV.Object.Name)

	almExamples, err := nfdCSV.GetAlmExamples()
	if err != nil {
		return nil, fmt.Errorf("failed to get ALM examples: %w", err)
	}

	almExamples, err = nfd.editAlmExample(almExamples)

	if err != nil {
		return nil, fmt.Errorf("failed to filter ALM examples: %w", err)
	}

	nfdBuilder := nodefeature.NewBuilderFromObjectString(nfd.APIClient, almExamples)

	nfdBuilder.Definition.Spec.TopologyUpdater = config.EnableTopology
	if config.WorkerConfig != "" {
		nfdBuilder.Definition.Spec.WorkerConfig.ConfigData = config.WorkerConfig
	}

	if config.Image != "" {
		nfdBuilder.Definition.Spec.Operand.Image = config.Image
		glog.V(nfd.LogLevel).Infof("Using custom NFD image: %s", config.Image)
	}

	return nfdBuilder, nil
}

// editAlmExample filters ALM examples to include only NodeFeatureDiscovery.
func (nfd *NFDCRUtils) editAlmExample(almExample string) (string, error) {
	var items []map[string]interface{}
	err := json.Unmarshal([]byte(almExample), &items)

	if err != nil {
		return "", fmt.Errorf("failed to unmarshal ALM examples JSON: %w", err)
	}

	var filtered []map[string]interface{}

	for _, item := range items {
		if kind, ok := item["kind"]; ok && kind == "NodeFeatureDiscovery" {
			filtered = append(filtered, item)
		}
	}

	if len(filtered) == 0 {
		return "", fmt.Errorf("no NodeFeatureDiscovery found in ALM examples")
	}

	output, err := json.MarshalIndent(filtered, "", "  ")

	if err != nil {
		return "", fmt.Errorf("failed to marshal filtered JSON: %w", err)
	}

	glog.V(nfd.LogLevel).Infof("Filtered ALM examples to %d NodeFeatureDiscovery objects", len(filtered))

	return string(output), nil
}
