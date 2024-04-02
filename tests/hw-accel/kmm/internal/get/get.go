package get

import (
	"encoding/base64"
	"fmt"

	"github.com/golang/glog"
	"github.com/hashicorp/go-version"

	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// NumberOfNodesForSelector returns the number or worker nodes.
func NumberOfNodesForSelector(apiClient *clients.Settings, selector map[string]string) (int, error) {
	nodeBuilder, err := nodes.List(apiClient, metav1.ListOptions{LabelSelector: labels.Set(selector).String()})

	if err != nil {
		fmt.Println("could not discover number of nodes")

		return 0, err
	}

	glog.V(kmmparams.KmmLogLevel).Infof("NumberOfNodesForSelector return %v nodes", len(nodeBuilder))

	return len(nodeBuilder), nil
}

// ClusterArchitecture returns first node architecture of the nodes that match nodeSelector (e.g. worker nodes).
func ClusterArchitecture(apiClient *clients.Settings, nodeSelector map[string]string) (string, error) {
	nodeLabel := "kubernetes.io/arch"

	return getLabelFromNodeSelector(apiClient, nodeLabel, nodeSelector)
}

// KernelFullVersion returns first node architecture of the nodes that match nodeSelector (e.g. worker nodes).
func KernelFullVersion(apiClient *clients.Settings, nodeSelector map[string]string) (string, error) {
	nodeBuilder, err := nodes.List(apiClient, metav1.ListOptions{LabelSelector: labels.Set(nodeSelector).String()})
	if err != nil {
		glog.V(kmmparams.KmmLogLevel).Infof("could not discover %v nodes", nodeSelector)

		return "", err
	}

	for _, node := range nodeBuilder {
		kernelVersion := node.Object.Status.NodeInfo.KernelVersion

		glog.V(kmmparams.KmmLogLevel).Infof("Found kernelVersion '%v'  on node '%v'",
			kernelVersion, node.Object.Name)

		return kernelVersion, nil
	}

	err = fmt.Errorf("could not find kernelVersion on node")

	return "", err
}

func getLabelFromNodeSelector(
	apiClient *clients.Settings,
	nodeLabel string,
	nodeSelector map[string]string) (string, error) {
	nodeBuilder, err := nodes.List(apiClient, metav1.ListOptions{LabelSelector: labels.Set(nodeSelector).String()})

	// Check if at least one node matching the nodeSelector has the specific nodeLabel label set to true
	// For example, look in all the worker nodes for specific label
	if err != nil {
		glog.V(kmmparams.KmmLogLevel).Infof("could not discover %v nodes", nodeSelector)

		return "", err
	}

	for _, node := range nodeBuilder {
		labelValue, ok := node.Object.Labels[nodeLabel]

		if ok {
			glog.V(kmmparams.KmmLogLevel).Infof("Found label '%v' with label value '%v' on node '%v'",
				nodeLabel, labelValue, node.Object.Name)

			return labelValue, nil
		}
	}

	err = fmt.Errorf("could not find one node with label '%s'", nodeLabel)

	return "", err
}

// MachineConfigPoolName returns machineconfigpool's name for a specified label.
func MachineConfigPoolName(apiClient *clients.Settings) string {
	nodeBuilder, err := nodes.List(
		apiClient,
		metav1.ListOptions{LabelSelector: labels.Set(map[string]string{"kubernetes.io": ""}).String()},
	)

	if err != nil {
		glog.V(kmmparams.KmmLogLevel).Infof("could not discover nodes")

		return ""
	}

	if len(nodeBuilder) == 1 {
		glog.V(kmmparams.KmmLogLevel).Infof("Using 'master' as mcp")

		return "master"
	}

	glog.V(kmmparams.KmmLogLevel).Infof("Using 'worker' as mcp")

	return "worker"
}

// SigningData returns struct used for creating secrets for module signing.
func SigningData(key string, value string) map[string][]byte {
	val, err := base64.StdEncoding.DecodeString(value)

	if err != nil {
		glog.V(kmmparams.KmmLogLevel).Infof("Error decoding signing key")
	}

	secretContents := map[string][]byte{key: val}

	return secretContents
}

// PreflightImage returns preflightvalidationocp image to be used based on architecture.
func PreflightImage(arch string) string {
	if arch == "arm64" {
		arch = "aarch64"
	}

	if arch == "amd64" {
		arch = "x86_64"
	}

	return fmt.Sprintf(kmmparams.PreflightTemplateImage, arch)
}

// PreflightReason returns the reason of a preflightvalidationocp check.
func PreflightReason(apiClient *clients.Settings, preflight, module, nsname string) (string, error) {
	pre, _ := kmm.PullPreflightValidationOCP(apiClient, preflight, nsname)

	preflightValidationOCP, err := pre.Get()

	if err == nil {
		reason := preflightValidationOCP.Status.CRStatuses[module].StatusReason
		glog.V(kmmparams.KmmLogLevel).Infof("VerificationStatus: %s", reason)

		return reason, nil
	}

	return "", err
}

// ModuleLoadedMessage returns message for a module loaded event.
func ModuleLoadedMessage(module, nsname string) string {
	message := fmt.Sprintf("Module %s/%s loaded into the kernel", nsname, module)
	glog.V(kmmparams.KmmLogLevel).Infof("Return: '%s'", message)

	return message
}

// ModuleUnloadedMessage returns message for a module unloaded event.
func ModuleUnloadedMessage(module, nsname string) string {
	message := fmt.Sprintf("Module %s/%s unloaded from the kernel", nsname, module)
	glog.V(kmmparams.KmmLogLevel).Infof("Return: '%s'", message)

	return message
}

// KmmOperatorVersion returns CSV version of the installed KMM operator.
func KmmOperatorVersion(apiClient *clients.Settings) (ver *version.Version, err error) {
	return operatorVersion(apiClient, "kernel", kmmparams.KmmOperatorNamespace)
}

// KmmHubOperatorVersion returns CSV version of the installed KMM-HUB operator.
func KmmHubOperatorVersion(apiClient *clients.Settings) (ver *version.Version, err error) {
	return operatorVersion(apiClient, "hub", kmmparams.KmmHubOperatorNamespace)
}

func operatorVersion(apiClient *clients.Settings, namePattern, namespace string) (ver *version.Version, err error) {
	csv, err := olm.ListClusterServiceVersionWithNamePattern(apiClient, namePattern,
		namespace)

	if err != nil {
		return nil, err
	}

	for _, c := range csv {
		glog.V(kmmparams.KmmLogLevel).Infof("CSV: %s, Version: %s, Status: %s",
			c.Object.Spec.DisplayName, c.Object.Spec.Version, c.Object.Status.Phase)

		csvVersion, _ := version.NewVersion(c.Object.Spec.Version.String())

		return csvVersion, nil
	}

	return nil, err
}
