package dpdkenv

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/nto" //nolint:misspell
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/dpdk/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netconfig"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netenv"
	v2 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/performanceprofile/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// DoesClusterSupportDpdkTests verifies if given cluster supports dpdk workload and test cases.
func DoesClusterSupportDpdkTests(
	apiClient *clients.Settings, netConfig *netconfig.NetworkConfig, requiredCPU int64, requestedRAMGb int) error {
	glog.V(90).Infof("Verifying if cluster supports dpdk tests")

	err := netenv.DoesClusterHasEnoughNodes(apiClient, netConfig, 1, 2)

	if err != nil {
		return err
	}

	workerNodeList, err := nodes.List(
		apiClient,
		metav1.ListOptions{LabelSelector: labels.Set(netConfig.WorkerLabelMap).String()},
	)

	if err != nil {
		return err
	}

	for _, worker := range workerNodeList {
		if int(worker.Object.Status.Capacity.Memory().Value()/1024/1024/1024) < requestedRAMGb {
			return fmt.Errorf("worker %s has less than required ram number: %d", worker.Object.Name, requestedRAMGb)
		}

		if worker.Object.Status.Capacity.Cpu().Value() < requiredCPU {
			return fmt.Errorf("worker %s has less than required cpu cores: %d", worker.Object.Name, requiredCPU)
		}
	}

	err = netenv.IsSriovDeployed(apiClient, netConfig)
	if err != nil {
		return err
	}

	return nil
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

		err = mcp.WaitToBeStableFor(time.Minute, tsparams.MCOWaitTimeout)

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

	return mcp.WaitToBeStableFor(time.Minute, tsparams.MCOWaitTimeout)
}
