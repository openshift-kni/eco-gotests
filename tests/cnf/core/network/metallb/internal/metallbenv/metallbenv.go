package metallbenv

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/metallb"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netinittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/internal/netparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/core/network/metallb/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

// DoesClusterSupportMetalLbTests verifies if given environment supports metalLb tests.
func DoesClusterSupportMetalLbTests(requiredCPNodeNumber, requiredWorkerNodeNumber int) error {
	glog.V(90).Infof("Verifying if MetalLb operator deployed")

	if err := isMetalLbDeployed(); err != nil {
		return err
	}

	workerNodeList, err := nodes.List(
		APIClient,
		metav1.ListOptions{LabelSelector: labels.Set(NetConfig.WorkerLabelMap).String()},
	)

	if err != nil {
		return err
	}

	glog.V(90).Infof("Verifying if cluster has enough workers to run MetalLb tests")

	if len(workerNodeList) < requiredWorkerNodeNumber {
		return fmt.Errorf("cluster has less than %d worker nodes", requiredWorkerNodeNumber)
	}

	controlPlaneNodeList, err := nodes.List(
		APIClient,
		metav1.ListOptions{LabelSelector: labels.Set(NetConfig.ControlPlaneLabelMap).String()},
	)

	if err != nil {
		return err
	}

	glog.V(90).Infof("Verifying if cluster has enough control-plane nodes to run MetalLb tests")

	if len(controlPlaneNodeList) < requiredCPNodeNumber {
		return fmt.Errorf("cluster has less than %d control-plane nodes", requiredCPNodeNumber)
	}

	return nil
}

// CreateNewMetalLbDaemonSetAndWaitUntilItsRunning creates or recreates the new metalLb daemonset and waits until
// daemonset is in Ready state.
func CreateNewMetalLbDaemonSetAndWaitUntilItsRunning(timeout time.Duration, nodeLabel map[string]string) error {
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
		APIClient, tsparams.MetalLbIo, NetConfig.MlbOperatorNamespace, nodeLabel)
	_, err = metalLbIo.Create()

	if err != nil {
		return err
	}

	var metalLbDs *daemonset.Builder

	err = wait.PollUntilContextTimeout(
		context.TODO(), 3*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			metalLbDs, err = daemonset.Pull(APIClient, tsparams.MetalLbDsName, NetConfig.MlbOperatorNamespace)
			if err != nil {
				glog.V(90).Infof("Error to pull daemonset %s namespace %s, retry",
					tsparams.MetalLbDsName, NetConfig.MlbOperatorNamespace)

				return false, nil
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

// DefineIterationParams defines ip settings for iteration based on ipFamily parameter.
func DefineIterationParams(
	ipv4AddrList,
	ipv6AddrList,
	nodeExtIPv4AddrList,
	nodeExtIPv6AddrList []string,
	ipFamily string,
) (
	masterClientPod,
	subnetMast string,
	mlbAddrList,
	nodeExtAddrList,
	addressPool,
	frrPodMasterIPs []string,
	err error) {
	switch ipFamily {
	case netparam.IPV4Family:
		return "172.16.0.1",
			netparam.IPSubnet24,
			ipv4AddrList,
			nodeExtIPv4AddrList,
			[]string{"3.3.3.1", "3.3.3.240"},
			[]string{"172.16.0.253", "172.16.0.254"},
			doesIPListsHaveEnoughAddresses(ipv4AddrList, nodeExtIPv4AddrList, ipFamily)

	case netparam.IPV6Family:
		return "2002:1:1::3",
			netparam.IPSubnet64,
			ipv6AddrList,
			nodeExtIPv6AddrList,
			[]string{"2002:2:2::1", "2002:2:2::5"},
			[]string{"2002:1:1::1", "2002:2:2::2"},
			doesIPListsHaveEnoughAddresses(ipv6AddrList, nodeExtIPv6AddrList, ipFamily)
	}

	return "", "", nil, nil, nil, nil, fmt.Errorf(fmt.Sprintf(
		"ipStack parameter is invalid allowed values are %s, %s ", netparam.IPV4Family, netparam.IPV6Family))
}

func doesIPListsHaveEnoughAddresses(mlbAddrList, nodeExtAddrList []string, ipFamily string) error {
	if len(mlbAddrList) < 2 {
		return fmt.Errorf(
			"env var ECO_CNF_CORE_NET_MLB_ADDR_LIST doesn't have enought addresses for %s interation", ipFamily)
	}

	if len(nodeExtAddrList) < 2 {
		return fmt.Errorf("cluster nodes don't have enought external addresses for %s interation", ipFamily)
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
