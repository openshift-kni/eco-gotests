package await

import (
	"fmt"
	"net"
	"os/exec"
	"time"

	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/deployment"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/pod"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/statefulset"

	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WaitUntilAllDeploymentsReady waits for the duration of the defined timeout or until all deployments
// in the namespace reach the Ready condition.
func WaitUntilAllDeploymentsReady(apiClient *clients.Settings, nsname string, timeout time.Duration) (bool, error) {
	deployments, err := deployment.List(apiClient, nsname, metav1.ListOptions{})

	if err != nil {
		glog.V(100).Infof("deployment list error: %s", err)

		return false, err
	}

	for _, testDeployment := range deployments {
		if !testDeployment.IsReady(timeout) {
			return false, fmt.Errorf("deployment %s not ready in time. available replicas: %d",
				testDeployment.Definition.Name, testDeployment.Object.Status.AvailableReplicas)
		}
	}

	return true, nil
}

// WaitUntilAllStatefulSetsReady waits for the duration of the defined timeout or until all deployments
// in the namespace reach the Ready condition.
func WaitUntilAllStatefulSetsReady(apiClient *clients.Settings, nsname string, timeout time.Duration) (bool, error) {
	statefulsets, err := statefulset.List(apiClient, nsname, metav1.ListOptions{})

	if err != nil {
		glog.V(100).Infof("statefulsets list error: %s", err)

		return false, err
	}

	for _, testStatefulset := range statefulsets {
		if !testStatefulset.IsReady(timeout) {
			return false, fmt.Errorf("statefulset %s not ready in time. available replicas: %d",
				testStatefulset.Definition.Name, testStatefulset.Object.Status.AvailableReplicas)
		}
	}

	return true, nil
}

// WaitUntilAllPodsReady waits for the duration of the defined timeout or until all deployments
// in the namespace reach the Ready condition.
func WaitUntilAllPodsReady(apiClient *clients.Settings, nsname string, timeout time.Duration) (bool, error) {
	pods, err := pod.List(apiClient, nsname, metav1.ListOptions{})

	if err != nil {
		glog.V(100).Infof("pods list error: %s", err)

		return false, err
	}

	for _, testPod := range pods {
		err = testPod.WaitUntilReady(timeout)
		if err != nil {
			glog.V(100).Infof("pod %s did not become ready in time: %s", testPod.Object.Name, err)

			return false, err
		}
	}

	return true, nil
}

// WaitUntilNodeIsUnreachable waits until a hostname is not reachble via its IPv4 or IPv6 address.
func WaitUntilNodeIsUnreachable(hostname string, timeout time.Duration) error {
	ip4Unreachable := false
	ip6Unreachable := false
	// Get the IPv4 and IPv6 addresses of the node
	ipv4Addr, err := net.ResolveIPAddr("ip4", hostname)
	if err != nil {
		ip4Unreachable = true
	}

	ipv6Addr, err := net.ResolveIPAddr("ip6", hostname)
	if err != nil {
		ip6Unreachable = true
	}

	// Set the timeout for the function
	deadline := time.Now().Add(timeout)

	// Wait for the node to become unreachable
	for {
		// Ping the node's IPv4 address
		if !ip4Unreachable {
			cmd := exec.Command("ping", "-c", "1", ipv4Addr.String())
			_, err := cmd.CombinedOutput()

			if err != nil {
				ip4Unreachable = true
			}
		}

		// Ping the node's IPv6 address
		if !ip6Unreachable {
			cmd := exec.Command("ping", "-c", "1", ipv6Addr.String())
			_, err = cmd.CombinedOutput()

			if err != nil {
				ip6Unreachable = true
			}
		}

		if (ip4Unreachable) && (ip6Unreachable) {
			glog.V(100).Infof("Node is unreachable.")

			return nil
		}

		// If both pings succeed, the node is still reachable
		glog.V(100).Infof("Node is still reachable. Waiting...")

		// Check if the deadline has been reached
		if time.Now().After(deadline) {
			// If the timeout is reached, the node is considered to be reachable
			return fmt.Errorf("node has not become unreachable in time")
		}

		// Wait for a second
		time.Sleep(time.Second)
	}
}
