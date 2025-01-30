package netenv

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nto"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	v2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// DoesClusterHasEnoughNodes verifies if given cluster has enough nodes to run tests.
func DoesClusterHasEnoughNodes(
	apiClient *clients.Settings,
	netConfig *netconfig.NetworkConfig,
	requiredCPNodeNumber int,
	requiredWorkerNodeNumber int) error {
	glog.V(90).Infof("Verifying if cluster has enough workers to run tests")

	workerNodeList, err := nodes.List(
		apiClient,
		metav1.ListOptions{LabelSelector: labels.Set(netConfig.WorkerLabelMap).String()},
	)

	if err != nil {
		return err
	}

	if len(workerNodeList) < requiredWorkerNodeNumber {
		return fmt.Errorf("cluster has less than %d worker nodes", requiredWorkerNodeNumber)
	}

	controlPlaneNodeList, err := nodes.List(
		apiClient,
		metav1.ListOptions{LabelSelector: labels.Set(netConfig.ControlPlaneLabelMap).String()},
	)

	if err != nil {
		return err
	}

	glog.V(90).Infof("Verifying if cluster has enough control-plane nodes to run tests")

	if len(controlPlaneNodeList) < requiredCPNodeNumber {
		return fmt.Errorf("cluster has less than %d control-plane nodes", requiredCPNodeNumber)
	}

	return nil
}

// IsSriovDeployed verifies that the sriov operator is deployed.
func IsSriovDeployed(apiClient *clients.Settings, netConfig *netconfig.NetworkConfig) error {
	sriovNS := namespace.NewBuilder(apiClient, netConfig.SriovOperatorNamespace)
	if !sriovNS.Exists() {
		return fmt.Errorf("error SR-IOV namespace %s doesn't exist", sriovNS.Definition.Name)
	}

	for _, sriovDaemonsetName := range netparam.OperatorSriovDaemonsets {
		sriovDaemonset, err := daemonset.Pull(
			apiClient, sriovDaemonsetName, netConfig.SriovOperatorNamespace)

		if err != nil {
			return fmt.Errorf("error to pull SR-IOV daemonset %s from the cluster", sriovDaemonsetName)
		}

		if !sriovDaemonset.IsReady(30 * time.Second) {
			return fmt.Errorf("error SR-IOV daemonset %s is not in ready/ready state",
				sriovDaemonsetName)
		}
	}

	return nil
}

// BFDHasStatus verifies that BFD session on a pod has given status.
func BFDHasStatus(frrPod *pod.Builder, bfdPeer string, status string) error {
	bfdStatusOut, err := frrPod.ExecCommand(append(netparam.VtySh, "sh bfd peers brief json"))
	if err != nil {
		return err
	}

	var result []netparam.BFDDescription

	err = json.Unmarshal(bfdStatusOut.Bytes(), &result)
	if err != nil {
		return err
	}

	for _, peer := range result {
		if peer.BFDPeer == bfdPeer && peer.BFDStatus != status {
			return fmt.Errorf("%s bfd status is %s (expected %s)", peer.BFDPeer, peer.BFDStatus, status)
		}
	}

	return nil
}

// MapFirstKeyValue returns the first key-value pair found in the input map.
// If the input map is empty, it returns empty strings.
func MapFirstKeyValue(inputMap map[string]string) (string, string) {
	for key, value := range inputMap {
		return key, value
	}

	return "", ""
}

// DeployPerformanceProfile installs performanceProfile on cluster.
func DeployPerformanceProfile(
	apiClient *clients.Settings,
	netConfig *netconfig.NetworkConfig,
	profileName string,
	isolatedCPU string,
	reservedCPU string,
	hugePages1GCount int32) error {
	glog.V(90).Infof("Ensuring cluster has correct PerformanceProfile deployed")

	mcp, err := mco.Pull(apiClient, netConfig.CnfMcpLabel)
	if err != nil {
		return fmt.Errorf("fail to pull MCP due to : %w", err)
	}

	performanceProfiles, err := nto.ListProfiles(apiClient)

	if err != nil {
		return fmt.Errorf("fail to list PerformanceProfile objects on cluster due to: %w", err)
	}

	if len(performanceProfiles) > 0 {
		for _, perfProfile := range performanceProfiles {
			if perfProfile.Object.Name == profileName {
				glog.V(90).Infof("PerformanceProfile %s exists", profileName)

				return nil
			}
		}

		glog.V(90).Infof("PerformanceProfile doesn't exist on cluster. Removing all pre-existing profiles")

		err := nto.CleanAllPerformanceProfiles(apiClient)

		if err != nil {
			return fmt.Errorf("fail to clean pre-existing performance profiles due to %w", err)
		}

		err = mcp.WaitToBeStableFor(time.Minute, netparam.MCOWaitTimeout)

		if err != nil {
			return err
		}
	}

	glog.V(90).Infof("Required PerformanceProfile doesn't exist. Installing new profile PerformanceProfile")

	_, err = nto.NewBuilder(apiClient, profileName, isolatedCPU, reservedCPU, netConfig.WorkerLabelMap).
		WithHugePages("1G", []v2.HugePage{{Size: "1G", Count: hugePages1GCount}}).Create()

	if err != nil {
		return fmt.Errorf("fail to deploy PerformanceProfile due to: %w", err)
	}

	return mcp.WaitToBeStableFor(time.Minute, netparam.MCOWaitTimeout)
}

// RemoveSriovConfigurationAndWaitForSriovAndMCPStable removes all SR-IOV networks
// and policies in SR-IOV operator namespace.
func RemoveSriovConfigurationAndWaitForSriovAndMCPStable() error {
	glog.V(90).Infof("Removing all SR-IOV networks and policies")

	err := RemoveAllSriovNetworks()
	if err != nil {
		glog.V(90).Infof("Failed to remove all SR-IOV networks")

		return err
	}

	err = RemoveAllPoliciesAndWaitForSriovAndMCPStable()
	if err != nil {
		glog.V(90).Infof("Failed to remove all SR-IOV policies")

		return err
	}

	return nil
}

// RemoveAllSriovNetworks removes all SR-IOV networks.
func RemoveAllSriovNetworks() error {
	glog.V(90).Infof("Removing all SR-IOV networks")

	sriovNs, err := namespace.Pull(netinittools.APIClient, netinittools.NetConfig.SriovOperatorNamespace)
	if err != nil {
		glog.V(90).Infof("Failed to pull SR-IOV operator namespace")

		return err
	}

	err = sriovNs.CleanObjects(
		netparam.DefaultTimeout,
		sriov.GetSriovNetworksGVR())
	if err != nil {
		glog.V(90).Infof("Failed to remove SR-IOV networks from SR-IOV operator namespace")

		return err
	}

	return nil
}

// RemoveAllPoliciesAndWaitForSriovAndMCPStable removes all  SriovNetworkNodePolicies and waits until
// SR-IOV and MCP become stable.
func RemoveAllPoliciesAndWaitForSriovAndMCPStable() error {
	glog.V(90).Infof("Deleting all SriovNetworkNodePolicies and waiting for SR-IOV and MCP become stable.")

	err := sriov.CleanAllNetworkNodePolicies(netinittools.APIClient, netinittools.NetConfig.SriovOperatorNamespace)
	if err != nil {
		return err
	}

	return WaitForSriovAndMCPStable(
		netinittools.APIClient, netparam.MCOWaitTimeout, time.Minute,
		netinittools.NetConfig.CnfMcpLabel, netinittools.NetConfig.SriovOperatorNamespace)
}

// WaitForMcpStable waits for the stability of the MCP with the given name.
func WaitForMcpStable(apiClient *clients.Settings, waitingTime, stableDuration time.Duration, mcpName string) error {
	mcp, err := mco.Pull(apiClient, mcpName)

	if err != nil {
		return fmt.Errorf("fail to pull mcp %s from cluster due to: %s", mcpName, err.Error())
	}

	err = mcp.WaitToBeStableFor(stableDuration, waitingTime)

	if err != nil {
		return fmt.Errorf("cluster is not stable: %s", err.Error())
	}

	return nil
}
