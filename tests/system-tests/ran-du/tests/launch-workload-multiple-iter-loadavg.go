package ran_du_system_test

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/shell"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ran-du/internal/randuinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ran-du/internal/randuparams"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ran-du/internal/randutestworkload"
)

var _ = Describe(
	"LaunchWorkloadMultipleIterations",
	Ordered,
	ContinueOnFailure,
	Label("LaunchWorkloadMultipleIterations"), func() {
		It("Launch workload multiple times", reportxml.ID("45698"), Label("LaunchWorkloadMultipleIterations"), func() {

			By("Launch workload")
			for iter := 0; iter < RanDuTestConfig.LaunchWorkloadIterations; iter++ {
				fmt.Printf("Launch workload iteration no. %d\n", iter)

				By("Clean up workload namespace")
				if namespace.NewBuilder(APIClient, RanDuTestConfig.TestWorkload.Namespace).Exists() {
					err := randutestworkload.CleanNameSpace(randuparams.DefaultTimeout, RanDuTestConfig.TestWorkload.Namespace)
					Expect(err).ToNot(HaveOccurred(), "Failed to clean workload test namespace objects")
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

				By("Waiting for all pods to become ready")
				_, err = await.WaitUntilAllPodsReady(APIClient, RanDuTestConfig.TestWorkload.Namespace, randuparams.DefaultTimeout)
				Expect(err).ToNot(HaveOccurred(), "pod not ready: %s", err)
			}

			By("Observe node load average while workload is running")
			cmdToExec := "cat /proc/loadavg"

			for n := 0; n < 30; n++ {
				cmdResult, err := cluster.ExecCmdWithStdout(APIClient, cmdToExec)
				Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)

				for _, stdout := range cmdResult {
					if len(strings.Fields(stdout)) > 0 {
						buf := strings.Fields(stdout)[0]
						floatBuf, err := strconv.ParseFloat(strings.ReplaceAll(strings.ReplaceAll(buf, "\r", ""), "\n", ""), 32)
						if err != nil {
							fmt.Println(err)
						}
						Expect(floatBuf).To(BeNumerically("<", randuparams.TestMultipleLaunchWorkloadLoadAvg),
							"error: node load average detected above 100: %f", floatBuf)
					}
				}

				time.Sleep(10 * time.Second)
			}

		})
		AfterAll(func() {
			By("Cleaning up test workload resources")
			err := randutestworkload.CleanNameSpace(randuparams.DefaultTimeout, RanDuTestConfig.TestWorkload.Namespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to clean workload test namespace objects")
		})
	})
