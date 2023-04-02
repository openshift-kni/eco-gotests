package metallbenv

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-gotests/pkg/daemonset"
	"github.com/openshift-kni/eco-gotests/pkg/deployment"
	"github.com/openshift-kni/eco-gotests/pkg/metallb"
	"github.com/openshift-kni/eco-gotests/pkg/namespace"
	"github.com/openshift-kni/eco-gotests/pkg/nodes"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	"k8s.io/apimachinery/pkg/util/wait"
)

// DoesClusterSupportMetalLbTests verifies if given environment supports metalLb tests.
func DoesClusterSupportMetalLbTests(requiredCPNodeNumber, requiredWorkerNodeNumber int) error {
	glog.V(90).Infof("Verifying if MetalLb operator deployed")

	if err := isMetalLbDeployed(); err != nil {
		return err
	}

	workerNodeList := nodes.NewBuilder(APIClient, NetConfig.WorkerLabelMap)

	glog.V(90).Infof("Verifying if cluster has enough workers to run MetalLb tests")

	err := workerNodeList.Discover()

	if err != nil {
		return err
	}

	if len(workerNodeList.Objects) < requiredWorkerNodeNumber {
		return fmt.Errorf("cluster has less than %d worker nodes", requiredWorkerNodeNumber)
	}

	controlPlaneNodeList := nodes.NewBuilder(APIClient, NetConfig.ControlPlaneLabelMap)

	glog.V(90).Infof("Verifying if cluster has enough control-plane nodes to run MetalLb tests")

	err = controlPlaneNodeList.Discover()

	if err != nil {
		return err
	}

	if len(controlPlaneNodeList.Objects) < requiredCPNodeNumber {
		return fmt.Errorf("cluster has less than %d control-plane nodes", requiredCPNodeNumber)
	}

	return nil
}

// CreateNewMetalLbDaemonSetAndWaitUntilItsRunning creates or recreates the new metalLb daemonset and waits until
// daemonset is in Ready state.
func CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(timeout time.Duration) error {
	glog.V(90).Infof("Verifying if metalLb daemonset is running")

	metalLbIo, err := metallb.Pull(APIClient, tsparams.MetalLbIo, NetConfig.MlbOperatorNamespace)

	if err == nil {
		glog.V(90).Infof("MetalLb daemonset is running. Removing daemonset.")

		_, err = metalLbIo.Delete()

		if err != nil {
			return err
		}
	}

	glog.V(90).Infof("Create new metalLb speaker's daemonSet.")

	metalLbIo = metallb.NewBuilder(
		APIClient, tsparams.MetalLbIo, NetConfig.MlbOperatorNamespace, NetConfig.WorkerLabelMap)
	_, err = metalLbIo.Create()

	if err != nil {
		return err
	}

	var metalLbDs *daemonset.Builder

	err = wait.PollImmediate(3*time.Second, timeout, func() (bool, error) {
		metalLbDs, err = daemonset.Pull(APIClient, tsparams.MetalLbDsName, NetConfig.MlbOperatorNamespace)
		if err != nil {
			return false, err
		}

		return true, nil
	})

	if err != nil {
		return err
	}

	glog.V(90).Infof("Waiting until the new metalLb daemonset is in Ready state.")

	if metalLbDs.IsReady(timeout) {
		return nil
	}

	return fmt.Errorf("metallb daemonSet is not ready")
}

// GetMetalLbIPByIPStack returns metalLb IP addresses  from env var typo:ECO_CNF_CORE_NET_MLB_ADDR_LIST
// sorted by IPStack.
func GetMetalLbIPByIPStack() ([]string, []string, error) {
	var ipv4IPList, ipv6IPList []string

	glog.V(90).Infof("Getting MetalLb virtual ip addresses.")

	metalLbIPList, err := NetConfig.GetMetalLbVirIP()

	if err != nil {
		return nil, nil, err
	}

	for _, ipAddress := range metalLbIPList {
		glog.V(90).Infof("Validate if ip address: %s is in the correct format", ipAddress)

		if net.ParseIP(ipAddress) == nil {
			return nil, nil, fmt.Errorf("not valid IP %s", ipAddress)
		}

		glog.V(90).Infof("Sort ip address: %s by ip stack", ipAddress)

		if strings.Contains(ipAddress, ":") {
			ipv6IPList = append(ipv6IPList, ipAddress)
		} else {
			ipv4IPList = append(ipv4IPList, ipAddress)
		}
	}

	return ipv4IPList, ipv6IPList, nil
}

// IsEnvVarMetalLbIPinNodeExtNetRange validates that the environmental IP variable
// is in the same IP range as the br-ex interface of the cluster under-test.
func IsEnvVarMetalLbIPinNodeExtNetRange(nodeExtAddresses, metalLbEnvIPv4, metalLbEnvIPv6 []string) error {
	// Checks that the ECO_CNF_CORE_NET_MLB_ADDR_LIST is in the range of the cluster br-ex interface.
	glog.V(90).Infof("Checking if node's external IP is in the same subnet with metalLb virtual IP.")

	if metalLbEnvIPv4 == nil && metalLbEnvIPv6 == nil {
		return fmt.Errorf("IPv4 and IPv6 address lists are empty please check your env var")
	}

	for _, nodeExtAddress := range nodeExtAddresses {
		ipaddr, subnet, err := net.ParseCIDR(nodeExtAddress)

		if err != nil {
			return err
		}

		switch ipaddr.To4() {
		case nil:
			if !isAddressInRange(subnet, metalLbEnvIPv6) {
				return fmt.Errorf("metalLb virtual address %s is not in node subnet %s", metalLbEnvIPv6, subnet)
			}
		default:
			if !isAddressInRange(subnet, metalLbEnvIPv4) {
				return fmt.Errorf("metalLb virtual address %s is not in node subnet %s", metalLbEnvIPv4, subnet)
			}
		}
	}

	return nil
}

func isMetalLbDeployed() error {
	metalLbNS := namespace.NewBuilder(APIClient, NetConfig.MlbOperatorNamespace)
	if !metalLbNS.Exists() {
		return fmt.Errorf("error metallb namespace %s doesn't exist", metalLbNS.Definition.Name)
	}

	metalLbController, err := deployment.Pull(
		APIClient, tsparams.OperatorControllerManager, NetConfig.MlbOperatorNamespace)

	if err != nil {
		return fmt.Errorf("error to pull metallb controller deployment %s from cluster", tsparams.OperatorControllerManager)
	}

	if !metalLbController.IsReady(30 * time.Second) {
		return fmt.Errorf("error metallb controller deployment %s is not in ready/ready state",
			tsparams.OperatorControllerManager)
	}

	metalLbWebhook, err := deployment.Pull(APIClient, tsparams.OperatorWebhook, NetConfig.MlbOperatorNamespace)

	if err != nil {
		return fmt.Errorf("error to pull webhook deployment object %s from cluster", tsparams.OperatorWebhook)
	}

	if !metalLbWebhook.IsReady(30 * time.Second) {
		return fmt.Errorf("error metallb webhook deployment %s is not in ready/ready state",
			tsparams.OperatorWebhook)
	}

	return nil
}

func isAddressInRange(subnet *net.IPNet, addresses []string) bool {
	allAddressInRange := true
	for _, address := range addresses {
		if !allAddressInRange {
			break
		}

		allAddressInRange = subnet.Contains(net.ParseIP(address))
	}

	return allAddressInRange
}