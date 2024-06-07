package ran_du_system_test

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/platform"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/shell"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ran-du/internal/randuinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ran-du/internal/randuparams"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ran-du/internal/randutestworkload"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func buildOutputLine(data map[string]string) string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, key := range keys {
		sb.WriteString(fmt.Sprintf("%s,%s,", key, data[key]))
	}
	line := strings.TrimSuffix(sb.String(), ",")

	currentTime := time.Now().Format(time.RFC3339)
	return fmt.Sprintf("%s,%s\n", currentTime, line)
}

func savePodsRestartsInNamespace(namespace string, outputFile string) error {
	podList, err := pod.List(APIClient, namespace, v1.ListOptions{})
	if err != nil {
		return err
	}

	file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return err
	}
	defer file.Close()

	allPodsInNamespace := make(map[string]string)
	for _, pod := range podList {
		totalRestarts := 0
		for _, containerStatus := range pod.Object.Status.ContainerStatuses {
			totalRestarts += int(containerStatus.RestartCount)
		}
		allPodsInNamespace[pod.Object.Name] = fmt.Sprintf("%d", totalRestarts)
	}

	entry := buildOutputLine(allPodsInNamespace)

	_, err = file.WriteString(entry)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}
	return nil
}

func savePolicyStatus(clusterName string, outputFile string) error {

	file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return err
	}
	defer file.Close()

	allPolicies, err := ocm.ListPoliciesInAllNamespaces(APIClient,
		runtimeclient.ListOptions{Namespace: clusterName})
	if err != nil {
		return fmt.Errorf("failed to get policies in %q NS: %v", clusterName, err)
	}

	allPoliciesStatus := make(map[string]string)
	for _, policy := range allPolicies {
		pName := policy.Definition.Name
		pState := string(policy.Object.Status.ComplianceState)
		allPoliciesStatus[pName] = pState
	}

	entry := buildOutputLine(allPoliciesStatus)
	_, err = file.WriteString(entry)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}
	return nil
}

func verifyStabilityStatusChange(file_path string) (bool, error) {

	file, err := os.OpenFile(file_path, os.O_RDONLY, 0)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return false, err
	}

	defer file.Close()

	var previousLine []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		columns := strings.Split(line, ",")

		if previousLine == nil {
			previousLine = columns
			continue
		}

		for i := 1; i < len(columns); i += 2 {
			if columns[i] != previousLine[i] || columns[i+1] != previousLine[i+1] {
				return true, fmt.Errorf("change detected in column %d:\n previous: %s - %s\n current: %s - %s",
					i+1, previousLine[i], previousLine[i+1], columns[i], columns[i+1])
			}
		}

		previousLine = columns
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
		return false, err
	}
	return false, nil
}

var _ = Describe(
	"Stability",
	Ordered,
	ContinueOnFailure,
	Label("Stability"), func() {
		var (
			clusterName string
		)
		BeforeAll(func() {
			By("Preparing workload")
			if namespace.NewBuilder(APIClient, RanDuTestConfig.TestWorkload.Namespace).Exists() {
				By("Deleting workload using shell method")
				_, err := shell.ExecuteCmd(RanDuTestConfig.TestWorkload.DeleteShellCmd)
				Expect(err).ToNot(HaveOccurred(), "Failed to delete workload")
			}

			if RanDuTestConfig.TestWorkload.CreateMethod == randuparams.TestWorkloadShellLaunchMethod {
				By("Launching workload using shell method")
				_, err := shell.ExecuteCmd(RanDuTestConfig.TestWorkload.CreateShellCmd)
				Expect(err).ToNot(HaveOccurred(), "Failed to launch workload")
			}

			By("Waiting for deployment replicas to become ready")
			_, err := await.WaitUntilAllDeploymentsReady(APIClient, RanDuTestConfig.TestWorkload.Namespace,
				randuparams.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "error while waiting for deployment to become ready")

			By("Waiting for statefulset replicas to become ready")
			_, err = await.WaitUntilAllStatefulSetsReady(APIClient, RanDuTestConfig.TestWorkload.Namespace,
				randuparams.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "error while waiting for statefulsets to become ready")

			By("Waiting for pods replicas to become ready")
			_, err = await.WaitUntilAllPodsReady(APIClient, RanDuTestConfig.TestWorkload.Namespace, randuparams.DefaultTimeout)
			Expect(err).ToNot(HaveOccurred(), "pod not ready: %s", err)

			By("Fetching Cluster name")
			clusterName, err = platform.GetOCPClusterName(APIClient)
			Expect(err).ToNot(HaveOccurred(), "Failed to get cluster name")

		})
		It("Stability", reportxml.ID("TBD"), Label("StabilityWorkload"), func() {

			outputDir := "/tmp"
			policiesOutputFile := fmt.Sprintf("%s/stability_policies.log", outputDir)
			namespaces := []string{"openshift-etcd", "openshift-apiserver"}

			totalDuration := 1 * time.Minute
			interval := 5 * time.Second
			startTime := time.Now()

			for time.Since(startTime) < totalDuration {

				savePolicyStatus(clusterName, policiesOutputFile)

				for _, namespace := range namespaces {
					savePodsRestartsInNamespace(namespace, fmt.Sprintf("%s/stability_%s.log", outputDir, namespace))
				}

				time.Sleep(interval)
			}

			// Final check of all values
			var stabilityErrors []string

			// Verify policies
			_, err := verifyStabilityStatusChange(policiesOutputFile)
			if err != nil {
				stabilityErrors = append(stabilityErrors, err.Error())
			}

			// Verify podRestarts
			for _, namespace := range namespaces {
				_, err := verifyStabilityStatusChange(fmt.Sprintf("%s/stability_%s.log", outputDir, namespace))
				if err != nil {
					stabilityErrors = append(stabilityErrors, err.Error())
				}
			}

			if len(stabilityErrors) > 0 {
				Expect(stabilityErrors).ToNot(HaveOccurred(), "One or more errors in stability tests")
			}

		})
		AfterAll(func() {
			By("Cleaning up test workload resources")
			err := randutestworkload.CleanNameSpace(randuparams.DefaultTimeout, RanDuTestConfig.TestWorkload.Namespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to clean workload test namespace objects")
		})
	})
