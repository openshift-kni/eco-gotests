package pods

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	amdparams "github.com/openshift-kni/eco-gotests/tests/hw-accel/amdgpu/params"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeLabellerPodsFromNodes - Get all Node Labeller Pods from the given nodes.
func NodeLabellerPodsFromNodes(apiClient *clients.Settings, nodes []*nodes.Builder) ([]*pod.Builder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), amdparams.DefaultTimeout*time.Second)
	defer cancel()

	var waitGroup sync.WaitGroup

	var nodeLabellerPodBuilders []*pod.Builder

	errCh := make(chan error, len(nodes)*amdparams.MaxNodeLabellerPodsPerNode)

	defer close(errCh)

	nodeLabellerPodNamePrefix := fmt.Sprintf("%s-node-labeller-", amdparams.DeviceConfigName)

	for _, node := range nodes {
		waitGroup.Add(1)

		go PodsFromNodeByPrefixWithTimeout(ctx, &waitGroup, errCh, apiClient, &nodeLabellerPodBuilders, node,
			nodeLabellerPodNamePrefix, amdparams.MaxNodeLabellerPodsPerNode, amdparams.DefaultTimeout*time.Second,
			amdparams.DefaultSleepInterval*time.Second)
	}

	waitGroup.Wait()

	select {
	case err := <-errCh:
		return nil, fmt.Errorf("got the following error while trying to get Node Labeller Pods: %w", err)
	default:
		return nodeLabellerPodBuilders, nil
	}
}

// PodsFromNodeByPrefixWithTimeout - Get all the Pods with the given Prefix on a node
// and store in 'podsResults *[]*pod.Builder'.
func PodsFromNodeByPrefixWithTimeout(ctx context.Context, waitGroup *sync.WaitGroup, errCh chan error,
	apiClient *clients.Settings, podsResults *[]*pod.Builder, node *nodes.Builder,
	prefix string, cnt int, timeout time.Duration, interval time.Duration) {
	defer waitGroup.Done()

	funcCtx, funcCtxCancel := context.WithTimeout(ctx, timeout)
	defer funcCtxCancel()

	podListFieldSelector := fmt.Sprintf("spec.nodeName=%s", node.Object.Name)

	for {
		select {
		case <-funcCtx.Done():
			errCh <- fmt.Errorf("timeout period has been exceeded while waiting "+
				"for pods with prefix '%s'on node %s", prefix, node.Object.Name)

			return

		case <-time.After(interval):
			var podsWithPrefix []*pod.Builder

			glog.V(amdparams.AMDGPULogLevel).Infof("Listing Pods on node %s", node.Object.Name)
			podBuilders, podsListErr := pod.List(apiClient, amdparams.AMDGPUNamespace,
				metav1.ListOptions{FieldSelector: podListFieldSelector})

			if podsListErr != nil {
				errCh <- fmt.Errorf("failed to list Pods on node '%s'.\n%w", node.Object.Name, podsListErr)

				return
			}

			for _, podBuilder := range podBuilders {
				if strings.HasPrefix(podBuilder.Object.Name, prefix) {
					podsWithPrefix = append(podsWithPrefix, podBuilder)
				}
			}

			if len(podsWithPrefix) > cnt {
				errCh <- fmt.Errorf("got too many Pods ('%d') with prefix of '%s' on node '%s'. "+
					"Maximum Pods allowed: '%d'", len(podsWithPrefix), prefix, node.Object.Name, cnt)

				return
			}

			if len(podsWithPrefix) == cnt {
				*podsResults = append(*podsResults, podsWithPrefix...)

				return
			}
		}
	}
}

// WaitUntilNoNodeLabellerPodes - Wait until no more Node Labeller Pods on AMD GPU Worker Nodes.
func WaitUntilNoNodeLabellerPodes(apiClient *clients.Settings) error {
	nodeLabellerPodNamePrefix := fmt.Sprintf("%s-node-labeller-", amdparams.DeviceConfigName)

	return WaitUntilNoMorePodsInNamespaceByNameWithTimeout(context.TODO(), apiClient, amdparams.AMDGPUNamespace,
		nodeLabellerPodNamePrefix, amdparams.DefaultTimeout*time.Second, amdparams.DefaultSleepInterval*time.Second)
}

// WaitUntilNoMorePodsInNamespaceByNameWithTimeout - Wait until no more pods with the given prefix on the cluster.
func WaitUntilNoMorePodsInNamespaceByNameWithTimeout(ctx context.Context, apiClient *clients.Settings,
	namespace string, prefix string, timeout time.Duration, chkInterval time.Duration) error {
	newCtx, cancelNewCtx := context.WithTimeout(ctx, timeout)
	defer cancelNewCtx()

	var podsWithPrefix []*pod.Builder

	for {
		select {
		case <-newCtx.Done():
			return fmt.Errorf("timeout period has been exceeded while waiting until no more Pods with prefix '%s'", prefix)
		case <-time.After(chkInterval):
			podsWithPrefix = nil
			listedPods, listPodsErr := pod.List(apiClient, namespace)

			if listPodsErr != nil {
				return fmt.Errorf("failed to list Pods. %w", listPodsErr)
			}

			for _, podBuilder := range listedPods {
				if strings.HasPrefix(podBuilder.Object.Name, prefix) {
					podsWithPrefix = append(podsWithPrefix, podBuilder)
				}
			}

			if len(podsWithPrefix) == 0 {
				return nil
			}
		}
	}
}
