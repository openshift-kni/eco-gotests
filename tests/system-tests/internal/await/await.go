package await

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/daemonset"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/systemtestsparams"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/statefulset"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
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

// WaitUntilDeploymentReady waits for the duration of the defined timeout or until specific deployment
// in the namespace reach the Ready condition.
func WaitUntilDeploymentReady(apiClient *clients.Settings, name, nsname string, timeout time.Duration) error {
	err := wait.PollUntilContextTimeout(
		context.TODO(),
		time.Second*2,
		timeout,
		true,
		func(ctx context.Context) (bool, error) {
			irDeployment, err := deployment.Pull(apiClient, name, nsname)

			if err != nil {
				return false, nil
			}

			isReady := irDeployment.IsReady(time.Second * 2)

			if isReady {
				return true, nil
			}

			return false, nil
		})

	if err != nil {
		irDeployment, err := deployment.Pull(apiClient, name, nsname)

		if err != nil {
			glog.V(100).Infof("deployment %s in namespace %s not exists; %w", name, nsname, err)

			return err
		}

		glog.V(100).Infof("deployment %s in namespace %s not ready in time. available replicas: %d",
			irDeployment.Definition.Name,
			irDeployment.Definition.Namespace,
			irDeployment.Object.Status.AvailableReplicas)

		return err
	}

	return nil
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

// WaitUntilNewMetalLbDaemonSetIsRunning waits until the new metalLb daemonset is in Ready state.
func WaitUntilNewMetalLbDaemonSetIsRunning(apiClient *clients.Settings, timeout time.Duration) error {
	glog.V(90).Infof("Verifying if metalLb daemonset %s is running in namespace %s",
		systemtestsparams.MetalLbDaemonSetName, systemtestsparams.MetalLbOperatorNamespace)

	var metalLbDs *daemonset.Builder

	var err error

	err = wait.PollUntilContextTimeout(
		context.TODO(), 3*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			metalLbDs, err = daemonset.Pull(apiClient,
				systemtestsparams.MetalLbDaemonSetName,
				systemtestsparams.MetalLbOperatorNamespace)
			if err != nil {
				glog.V(90).Infof("Error to pull daemonset %s namespace %s, retry",
					systemtestsparams.MetalLbDaemonSetName, systemtestsparams.MetalLbOperatorNamespace)

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

// WaitForThePodReplicasCountInNamespace waiting for the specific pod replicas count in
// the namespace that match options.
func WaitForThePodReplicasCountInNamespace(
	apiClient *clients.Settings,
	nsname string,
	options metav1.ListOptions,
	replicasCount int,
	timeout time.Duration,
) (bool, error) {
	glog.V(100).Infof("Waiting for %d pod replicas count in namespace %s with options %v"+
		" are in running state", replicasCount, nsname, options)

	if nsname == "" {
		glog.V(100).Infof("'nsname' parameter can not be empty")

		return false, fmt.Errorf("failed to list pods, 'nsname' parameter is empty")
	}

	err := wait.PollUntilContextTimeout(
		context.TODO(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			podsList, err := pod.List(apiClient, nsname, options)
			if err != nil {
				glog.V(100).Infof("Failed to list all pods due to %s", err.Error())

				return false, nil
			}

			if len(podsList) != replicasCount {
				glog.V(100).Infof("pod replicas count not equal to the expected: "+
					"current %d, expected: %d", len(podsList), replicasCount)

				return false, nil
			}

			return true, nil
		})

	if err != nil {
		return false, err
	}

	return true, nil
}
