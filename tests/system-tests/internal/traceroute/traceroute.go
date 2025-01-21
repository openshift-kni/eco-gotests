package traceroute

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/apiobjectshelper"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	scc "github.com/openshift-kni/eco-gotests/tests/system-tests/internal/scc"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	trDeployRBACName = "privileged-rdscore-traceroute"
	trDeploySAName   = "default"
	trRBACRole       = "system:openshift:scc:privileged"
)

// CreateTraceRouteDeployment creates packet traceroute pods on the nodes specified in scheduleOnNodes.
func CreateTraceRouteDeployment(
	apiClient *clients.Settings,
	trPodNamespace,
	trDeploymentLabel,
	trImage string,
	scheduleOnNodes []string) (*deployment.Builder, error) {
	glog.V(100).Infof("Getting the image and tag for tools from the release image")

	trDeploymentName := fmt.Sprintf("%s-packet-traceroute", trPodNamespace)

	glog.V(100).Infof("Create traceroute namespace %s", trPodNamespace)

	err := ensureNamespaceExists(apiClient, trPodNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create traceroute namespace %s: %w", trPodNamespace, err)
	}

	glog.V(100).Infof("Adding SCC privileged to the traceroute namespace")

	err = scc.AddPrivilegedSCCtoDefaultSA(trPodNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to add SCC privileged to the traceroute namespace %s: %w",
			trPodNamespace, err)
	}

	trDeployment, err := createTraceRouteDeployment(
		apiClient,
		trImage,
		trDeploymentName,
		trPodNamespace,
		trDeploymentLabel,
		scheduleOnNodes,
	)

	if err != nil {
		return trDeployment, err
	}

	retries := 3
	pollInterval := 5

	for i := 0; i < retries; i++ {
		trDeployment, err = deployment.Pull(apiClient, trDeploymentName, trPodNamespace)
		if err != nil {
			return nil, err
		}

		if trDeployment.Object.Status.ReadyReplicas == trDeployment.Object.Status.Replicas {
			return trDeployment, nil
		}

		time.Sleep(time.Duration(pollInterval) * time.Second)
	}

	return trDeployment, fmt.Errorf("deployment still not ready after %d tries", retries)
}

func ensureNamespaceExists(apiClient *clients.Settings, nsName string) error {
	glog.V(100).Infof("Create namespace %q", nsName)

	createNs := namespace.NewBuilder(apiClient, nsName)

	if createNs.Exists() {
		err := createNs.Delete()

		if err != nil {
			glog.V(100).Infof("Failed to delete namespace %q: %v", nsName, err)

			return fmt.Errorf("failed to delete namespace %q: %w", nsName, err)
		}

		err = wait.PollUntilContextTimeout(
			context.TODO(),
			time.Second,
			30*time.Second,
			true,
			func(ctx context.Context) (bool, error) {
				if createNs.Exists() {
					glog.V(100).Infof("Error deleting namespace %q", nsName)

					return false, nil
				}

				glog.V(100).Infof("Deleted namespace %q", createNs.Definition.Name)

				return true, nil
			})

		if err != nil {
			return fmt.Errorf("failed to delete traceroute namespace %q : %w", nsName, err)
		}
	}

	createNs, err := createNs.Create()

	if err != nil {
		glog.V(100).Infof("Error creating namespace %q: %v", nsName, err)

		return fmt.Errorf("failed to create namespace %q: %w", nsName, err)
	}

	err = wait.PollUntilContextTimeout(
		context.TODO(),
		time.Second,
		3*time.Second,
		true,
		func(ctx context.Context) (bool, error) {
			if !createNs.Exists() {
				glog.V(100).Infof("Error creating namespace %q", nsName)

				return false, nil
			}

			glog.V(100).Infof("Created namespace %q", createNs.Definition.Name)

			return true, nil
		})

	if err != nil {
		return fmt.Errorf("traceroute namespace %q not found created: %w", nsName, err)
	}

	return nil
}

// createTraceRouteDeployment creates a host networked pod in namespace <namespace> on node <nodeName>.
// It will start a packet traceroute, and it will log all GET request's source IP and the actual request string.
//
//nolint:funlen
func createTraceRouteDeployment(
	apiClient *clients.Settings,
	trImage,
	trDeploymentName,
	trNamespace,
	trDeploymentLabel string,
	scheduleOnHosts []string) (*deployment.Builder, error) {
	glog.V(100).Infof("Creating the traceroute deployment with image %s", trImage)

	var err error

	glog.V(100).Infof("Checking traceroute deployment don't exist")

	err = apiobjectshelper.DeleteDeployment(apiClient, trDeploymentName, trNamespace)

	if err != nil {
		return nil, fmt.Errorf("failed to delete deployment %s from namespace %s",
			trDeploymentName, trNamespace)
	}

	glog.V(100).Infof("Sleeping 10 seconds")
	time.Sleep(10 * time.Second)

	glog.V(100).Infof("Removing ServiceAccount")

	err = apiobjectshelper.DeleteServiceAccount(apiClient, trDeploySAName, trNamespace)

	if err != nil {
		return nil, fmt.Errorf("failed to remove serviceAccount %q from namespace %q",
			trDeploySAName, trNamespace)
	}

	glog.V(100).Infof("Creating ServiceAccount")

	err = apiobjectshelper.CreateServiceAccount(apiClient, trDeploySAName, trNamespace)

	if err != nil {
		return nil, fmt.Errorf("failed to create serviceAccount %q in namespace %q",
			trDeploySAName, trNamespace)
	}

	glog.V(100).Infof("Removing Cluster RBAC")

	err = apiobjectshelper.DeleteClusterRBAC(apiClient, trDeployRBACName)

	if err != nil {
		return nil, fmt.Errorf("failed to delete traceroute RBAC %q", trDeployRBACName)
	}

	glog.V(100).Infof("Creating Cluster RBAC")

	err = apiobjectshelper.CreateClusterRBAC(apiClient, trDeployRBACName, trRBACRole,
		trDeploySAName, trNamespace)

	if err != nil {
		return nil, fmt.Errorf("failed to create traceroute RBAC %q in namespace %s",
			trDeployRBACName, trNamespace)
	}

	glog.V(100).Infof("Defining container configuration")

	deployContainer := defineTraceRouteContainer(trImage)

	glog.V(100).Infof("Obtaining container definition")

	deployContainerCfg, err := deployContainer.GetContainerCfg()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain container definition: %w", err)
	}

	glog.V(100).Infof("Defining deployment configuration")

	deployLabelsMap := map[string]string{
		strings.Split(trDeploymentLabel, "=")[0]: strings.Split(trDeploymentLabel, "=")[1]}

	trDeployment := defineDeployment(
		apiClient,
		deployContainerCfg,
		trDeploymentName,
		trNamespace,
		trDeploySAName,
		scheduleOnHosts,
		deployLabelsMap)

	glog.V(100).Infof("Creating deployment")

	trDeployment, err = trDeployment.CreateAndWaitUntilReady(5 * time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment %s in namespace %s: %w",
			trDeploymentName, trNamespace, err)
	}

	if trDeployment == nil {
		return nil, fmt.Errorf("failed to create deployment %s", trDeploymentName)
	}

	return trDeployment, err
}

// sendProbesAndCheckOutput sends traceroute requests and makes sure that the expected string was seen in the output.
func sendProbesAndCheckOutput(
	trPod *pod.Builder,
	targetIP,
	targetPort,
	searchString string) (bool, error) {
	glog.V(100).Infof("Sending requests to the IP %s port %s from the pod %s/%s",
		targetIP, targetPort, trPod.Definition.Namespace, trPod.Definition.Name)

	cmdToRun := []string{"/bin/bash", "-c", fmt.Sprintf("traceroute -p %s %s", targetPort, targetIP)}

	var output bytes.Buffer

	var err error

	timeout := time.Minute
	err = wait.PollUntilContextTimeout(
		context.TODO(),
		time.Second*15,
		timeout,
		true,
		func(ctx context.Context) (bool, error) {
			output, err = trPod.ExecCommand(cmdToRun, trPod.Object.Spec.Containers[0].Name)

			if err != nil {
				glog.V(100).Infof("query failed. Request: %s, Output: %q, Error: %v",
					targetIP, output, err)

				return false, nil
			}

			glog.V(100).Infof("Successfully executed command from within a pod %q: %v",
				trPod.Object.Name, cmdToRun)
			glog.V(100).Infof("Command's output:\n\t%v", output.String())

			glog.V(100).Infof("Make sure that search string %s was seen in response %q",
				searchString, output.String())

			if output.String() == "" {
				return false, nil
			}

			if !strings.Contains(output.String(), searchString) {
				return false, nil
			}

			glog.V(100).Infof("Expected string %s was found in the command's output:\n\t%v",
				searchString, output.String())

			return true, nil
		})

	if err != nil {
		return false, fmt.Errorf("expected string %s not found in traceroute output: %q",
			searchString, output.String())
	}

	return true, nil
}

// SendTrafficFindExpectedString sends requests to the specific destination and makes sure that
// expected string was seen in command output.
func SendTrafficFindExpectedString(
	trPodObj *pod.Builder,
	targetIP,
	targetPort,
	searchString string) error {
	timeout := 3 * time.Minute
	err := wait.PollUntilContextTimeout(
		context.TODO(),
		3*time.Second,
		timeout,
		true,
		func(ctx context.Context) (bool, error) {
			result, err := sendProbesAndCheckOutput(
				trPodObj,
				targetIP,
				targetPort,
				searchString)

			if err == nil && result {
				return true, nil
			}

			return false, nil
		})

	if err != nil {
		return fmt.Errorf("expected string was not found in the traceroute output; %w", err)
	}

	return nil
}

func defineTraceRouteContainer(cImage string) *pod.ContainerBuilder {
	cName := "traceroute"

	cCmd := []string{"/bin/bash", "-c", "sleep INF"}

	glog.V(100).Infof("Creating container %q", cName)

	deployContainer := pod.NewContainerBuilder(cName, cImage, cCmd)

	glog.V(100).Infof("Defining SecurityContext")

	var trueFlag = true

	userUID := new(int64)

	*userUID = 0

	securityContext := &corev1.SecurityContext{
		RunAsUser:  userUID,
		Privileged: &trueFlag,
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		Capabilities: &corev1.Capabilities{
			Add: []corev1.Capability{
				"SETFCAP",
				"CAP_NET_RAW",
				"CAP_NET_ADMIN",
			},
		},
	}

	glog.V(100).Infof("Setting SecurityContext")

	deployContainer = deployContainer.WithSecurityContext(securityContext)

	glog.V(100).Infof("Dropping ALL security capability")

	deployContainer = deployContainer.WithDropSecurityCapabilities([]string{"ALL"}, true)

	glog.V(100).Infof("Enable TTY and Stdin; needed for immediate log propagation")

	deployContainer = deployContainer.WithTTY(true).WithStdin(true)

	glog.V(100).Infof("%q container's  definition:\n%#v", cName, deployContainer)

	return deployContainer
}

func defineDeployment(
	apiClient *clients.Settings,
	containerConfig *corev1.Container,
	deployName, deployNs, saName string,
	scheduleOnHosts []string,
	deployLabels map[string]string) *deployment.Builder {
	glog.V(100).Infof("Defining deployment %q in %q ns", deployName, deployNs)

	nodeAffinity := corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: corev1.NodeSelectorOpIn,
								Values:   scheduleOnHosts,
							},
						},
					},
				},
			},
		},
	}

	trDeployment := deployment.NewBuilder(apiClient, deployName, deployNs, deployLabels, *containerConfig)

	glog.V(100).Infof("Assigning ServiceAccount %q to the deployment", saName)

	trDeployment = trDeployment.WithServiceAccountName(saName)

	glog.V(100).Infof("Setting Replicas count")

	replicasCnt := len(scheduleOnHosts)

	trDeployment = trDeployment.WithReplicas(int32(replicasCnt))

	trDeployment = trDeployment.WithHostNetwork(true).WithAffinity(&nodeAffinity)

	return trDeployment
}
