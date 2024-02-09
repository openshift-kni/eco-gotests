package spk_system_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/url"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/spk/internal/spkinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/spk/internal/spkparams"
)

var _ = Describe(
	"SPK DNS/NAT46",
	Ordered,
	ContinueOnFailure,
	Label(spkparams.LabelSPKDnsNat46), func() {
		BeforeAll(func() {

			By(fmt.Sprintf("Asserting namespace %q exists", SPKConfig.SPKDataNS))

			_, err := namespace.Pull(APIClient, SPKConfig.SPKDataNS)
			Expect(err).To(BeNil(), fmt.Sprintf("Test namespace %s does not exist", SPKConfig.SPKDataNS))

			By(fmt.Sprintf("Asserting namespace %q exists", SPKConfig.SPKDnsNS))

			_, err = namespace.Pull(APIClient, SPKConfig.SPKDnsNS)
			Expect(err).To(BeNil(), fmt.Sprintf("Test namespace %s does not exist", SPKConfig.SPKDnsNS))

			By(fmt.Sprintf("Asserting namespace %q exists", SPKConfig.Namespace))

			_, err = namespace.Pull(APIClient, SPKConfig.Namespace)
			Expect(err).To(BeNil(), fmt.Sprintf("Test namespace %s does not exist", SPKConfig.Namespace))
		})

		It("Asserts DNS resoulution from new deployment", polarion.ID("63171"), Label("spkdns46new"), func(ctx SpecContext) {
			By("Asserting deployment exists")
			deploy, _ := deployment.Pull(APIClient, SPKConfig.WorkloadDeploymentName, SPKConfig.Namespace)

			if deploy != nil {
				glog.V(spkparams.SPKLogLevel).Infof("Existing deployment %q found. Removing...", SPKConfig.WorkloadDeploymentName)
				err := deploy.DeleteAndWait(300 * time.Second)
				Expect(err).ToNot(HaveOccurred(),
					fmt.Sprintf("failed to delete deployment %q", SPKConfig.WorkloadDeploymentName))
			}

			By("Checking that pods backed by deployment are gone")

			labels := "systemtest-test=spk-dns-nat46"

			glog.V(spkparams.SPKLogLevel).Infof("Checking for pods with label(s) %q \n", labels)

			Eventually(func() bool {
				oldPods, _ := pod.List(APIClient, SPKConfig.Namespace,
					metav1.ListOptions{LabelSelector: labels})

				return len(oldPods) == 0

			}, 6*time.Minute, 3*time.Second).WithContext(ctx).Should(BeTrue(), "pods matching label(s) still present")

			By("Defining container configuration")
			// NOTE(yprokule): image has entry point that does not require command to be set.
			deployContainer := pod.NewContainerBuilder("qe-spk", SPKConfig.WorkloadContainerImage,
				[]string{"/bin/bash", "-c", "sleep infinity"})

			By("Setting SCC")
			deployContainer = deployContainer.WithSecurityContext(&v1.SecurityContext{RunAsGroup: nil, RunAsUser: nil})

			By("Obtaining container definition")
			deployContainerCfg, err := deployContainer.GetContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "Failed to get container config")

			By("Defining deployment configuration")
			deploy = deployment.NewBuilder(APIClient, SPKConfig.WorkloadDeploymentName, SPKConfig.Namespace,
				map[string]string{"systemtest-test": "spk-dns-nat46", "systemtest-component": "spk"}, deployContainerCfg)

			By("Creating deployment")
			_, err = deploy.CreateAndWaitUntilReady(300 * time.Second)
			Expect(err).ToNot(HaveOccurred(), "failed to create deployment")

			By("Finding pod backed by deployment")
			appPods, err := pod.List(APIClient, SPKConfig.Namespace,
				metav1.ListOptions{LabelSelector: "systemtest-test=spk-dns-nat46"})
			Expect(err).ToNot(HaveOccurred(), "Failed to find pod with label")

			for _, _pod := range appPods {
				By("Running DNS resolution")

				cmdDig := fmt.Sprintf("dig -t A %s", SPKConfig.WorkloadTestURL)
				output, err := _pod.ExecCommand([]string{"/bin/sh", "-c", cmdDig}, "qe-spk")

				glog.V(spkparams.SPKLogLevel).Infof("Command's Output:\n%v\n", output.String())
				Expect(err).ToNot(HaveOccurred(), "Error performing DNS lookup from within pod")

				By("Fetching URL from outside the cluster")
				targetURL := fmt.Sprintf("http://%s:%s", SPKConfig.WorkloadTestURL, SPKConfig.WorkloadTestPort)
				cmd := fmt.Sprintf("curl -Ls --max-time 5 -o /dev/null -w '%%{http_code}' %s", targetURL)

				output, err = _pod.ExecCommand([]string{"/bin/sh", "-c", cmd}, "qe-spk")
				glog.V(spkparams.SPKLogLevel).Infof("Command's Output:\n%v\n", output.String())

				Expect(strings.TrimSpace(strings.Trim(output.String(), "'"))).To(BeElementOf("200", "404"))
				Expect(err).ToNot(HaveOccurred(), "Error fetching URL from within pod")
			}

		})

		It("Asserts DNS resolution from existing deployment", polarion.ID("66091"), Label("spkdns46existing"), func() {

			By("Asserting deployment exists")
			deploy, _ := deployment.Pull(APIClient, SPKConfig.WorkloadDCIDeploymentName, SPKConfig.Namespace)

			if deploy == nil {
				Skip(fmt.Sprintf("Deployment %q not found in %q ns",
					SPKConfig.WorkloadDeploymentName,
					SPKConfig.Namespace))
			}

			By("Finding pod backed by deployment")
			appPods, err := pod.List(APIClient, SPKConfig.Namespace,
				metav1.ListOptions{LabelSelector: "app=mttool"})
			Expect(err).ToNot(HaveOccurred(), "Failed to find DCI pod(s) matching label")

			for _, _pod := range appPods {
				By("Running DNS resolution from within a pod")

				cmdDig := fmt.Sprintf("dig -t A %s", SPKConfig.WorkloadTestURL)
				output, err := _pod.ExecCommand([]string{"/bin/sh", "-c", cmdDig}, "network-multitool")

				glog.V(spkparams.SPKLogLevel).Infof("Command's Output:\n%v\n", output.String())
				Expect(err).ToNot(HaveOccurred(), "Error performing DNS lookup from within pod")

				By("Access URL outside the cluster")

				targetURL := fmt.Sprintf("http://%s:%s", SPKConfig.WorkloadTestURL, SPKConfig.WorkloadTestPort)
				cmd := fmt.Sprintf("curl -Ls --max-time 5 -o /dev/null -w '%%{http_code}' %s", targetURL)

				output, err = _pod.ExecCommand([]string{"/bin/sh", "-c", cmd}, "network-multitool")
				glog.V(spkparams.SPKLogLevel).Infof("Command's Output:\n%v\n", output.String())

				Expect(strings.TrimSpace(strings.Trim(output.String(), "'"))).To(BeElementOf("200", "404"))
				Expect(err).ToNot(HaveOccurred(), "Error fetching external URL from within pod")
			}
		})

		It("Asserts DNS Resolution after SPK scale-down and scale-up", Label("spkscaledown"), func(ctx SpecContext) {
			By(fmt.Sprintf("Asserting APP deployment %q exists in %q namespace",
				SPKConfig.WorkloadDCIDeploymentName, SPKConfig.Namespace))

			deployApp, err := deployment.Pull(APIClient, SPKConfig.WorkloadDCIDeploymentName, SPKConfig.Namespace)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get deployment %q in %q namespace",
				SPKConfig.WorkloadDCIDeploymentName, SPKConfig.Namespace))

			By("Finding pod backed by deployment")
			appPods, err := pod.List(APIClient, SPKConfig.Namespace,
				metav1.ListOptions{LabelSelector: "app=mttool"})
			Expect(err).ToNot(HaveOccurred(), "Failed to list DCI pod(s) matching label")
			Expect(len(appPods)).ToNot(Equal(0), "Not enough application pods found")

			By("Asserting deployment exists")
			deployData, err := deployment.Pull(APIClient, SPKConfig.SPKDataTMMDeployName, SPKConfig.SPKDataNS)

			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get deployment %q in %q namespace",
				SPKConfig.SPKDataTMMDeployName, SPKConfig.SPKDataNS))

			By("Setting replicas count to 0")
			deployData = deployData.WithReplicas(0)

			By("Updating deployment")
			deployData, err = deployData.Update()

			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to scale down deployment %q in %q namespace",
				deployData.Object.Name, deployData.Object.Namespace))

			By(fmt.Sprintf("Wating for all pods from %q deployment in %q namespace to go away",
				deployData.Definition.Name, deployData.Definition.Namespace))

			Eventually(func() bool {
				appPods, err := pod.List(APIClient, SPKConfig.SPKDataNS,
					metav1.ListOptions{LabelSelector: "app=f5-tmm"})

				if err != nil {
					return false
				}

				return len(appPods) == 0

			}).WithContext(ctx).WithPolling(3 * time.Second).WithTimeout(5 * time.Minute).Should(BeTrue())

			By(fmt.Sprintf("Asserting deployment %q exists in %q namespace",
				SPKConfig.SPKDataTMMDeployName, SPKConfig.SPKDnsNS))

			deployDNS, err := deployment.Pull(APIClient, SPKConfig.SPKDataTMMDeployName, SPKConfig.SPKDnsNS)

			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to get deployment %q in %q namespace",
				SPKConfig.SPKDataTMMDeployName, SPKConfig.SPKDnsNS))

			By("Setting replicas count to 0")
			deployDNS = deployDNS.WithReplicas(0)

			By(fmt.Sprintf("Updating deployment %q exists in %q namespace",
				SPKConfig.SPKDataTMMDeployName, SPKConfig.SPKDnsNS))

			deployDNS, err = deployDNS.Update()

			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to scale down deployment %q in %q namespace",
				deployDNS.Object.Name, deployDNS.Object.Namespace))

			By(fmt.Sprintf("Wating for all pods from %q deployment in %q namespace to go away",
				deployDNS.Definition.Name, deployDNS.Definition.Namespace))

			Eventually(func() bool {
				appPods, err := pod.List(APIClient, SPKConfig.SPKDnsNS,
					metav1.ListOptions{LabelSelector: "app=f5-tmm"})

				if err != nil {
					return false
				}

				return len(appPods) == 0

			}).WithContext(ctx).WithPolling(3 * time.Second).WithTimeout(5 * time.Minute).Should(BeTrue())

			//
			// Scale up in spk-data namespace
			//
			By(fmt.Sprintf("Scale up %q deployment in %q namespace",
				deployData.Definition.Name, deployData.Definition.Namespace))

			Eventually(func() bool {

				glog.V(spkparams.SPKLogLevel).Infof("Pulling deployment %q in %q namespace",
					deployData.Definition.Name, deployData.Definition.Namespace)
				deployData, err = deployment.Pull(APIClient, SPKConfig.SPKDataTMMDeployName, SPKConfig.SPKDataNS)

				if err != nil {
					return false
				}

				glog.V(spkparams.SPKLogLevel).Infof("Setting replica count for  deployment %q in %q namespace",
					deployData.Definition.Name, deployData.Definition.Namespace)
				deployData = deployData.WithReplicas(2)

				glog.V(spkparams.SPKLogLevel).Infof("Updating deployment %q in %q namespace",
					deployData.Definition.Name, deployData.Definition.Namespace)
				deployData, err = deployData.Update()

				if err != nil {
					glog.V(spkparams.SPKLogLevel).Infof("Failed to update deployment %q in %q namespace: %v",
						deployData.Definition.Name, deployData.Definition.Namespace, err)

					return false
				}

				glog.V(spkparams.SPKLogLevel).Infof("Successfully updated deployment %q in %q namespace",
					deployData.Definition.Name, deployData.Definition.Namespace)

				return true
			}).WithContext(ctx).WithPolling(3*time.Second).WithTimeout(3*time.Minute).Should(BeTrue(),
				fmt.Sprintf("Failed to scale up %q in %q namespace", deployData.Definition.Name, deployData.Definition.Namespace))

			glog.V(spkparams.SPKLogLevel).Infof("Waiting for deployment %q in %q namespace to become Available",
				deployData.Definition.Name, deployData.Definition.Namespace)

			err = deployData.WaitUntilCondition("Available", 10*time.Minute)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Deployment %q in %q namespace hasn't reached Available state",
					deployData.Definition.Name, deployData.Definition.Namespace))

			//
			// Scale up in spk-dns namespace
			//
			By(fmt.Sprintf("Scale up %q deployment in %q namespace",
				deployDNS.Definition.Name, deployDNS.Definition.Namespace))

			Eventually(func() bool {

				glog.V(spkparams.SPKLogLevel).Infof("Pulling deployment %q in %q namespace",
					deployDNS.Definition.Name, deployDNS.Definition.Namespace)
				deployDNS, err = deployment.Pull(APIClient, SPKConfig.SPKDnsTMMDeployName, SPKConfig.SPKDnsNS)

				if err != nil {
					return false
				}

				glog.V(spkparams.SPKLogLevel).Infof("Setting replica count for  deployment %q in %q namespace",
					deployDNS.Definition.Name, deployDNS.Definition.Namespace)
				deployDNS = deployDNS.WithReplicas(2)

				glog.V(spkparams.SPKLogLevel).Infof("Updating deployment %q in %q namespace",
					deployDNS.Definition.Name, deployDNS.Definition.Namespace)
				deployDNS, err = deployDNS.Update()

				if err != nil {
					glog.V(spkparams.SPKLogLevel).Infof("Failed to update deployment %q in %q namespace: %v",
						deployDNS.Definition.Name, deployDNS.Definition.Namespace, err)

					return false
				}

				glog.V(spkparams.SPKLogLevel).Infof("Successfully updated deployment %q in %q namespace",
					deployDNS.Definition.Name, deployDNS.Definition.Namespace)

				return true
			}).WithContext(ctx).WithPolling(3*time.Second).WithTimeout(3*time.Minute).Should(BeTrue(),
				fmt.Sprintf("Failed to scale up %q in %q namespace", deployDNS.Definition.Name, deployDNS.Definition.Namespace))

			glog.V(spkparams.SPKLogLevel).Infof("Waiting for deployment %q in %q namespace to become Available",
				deployDNS.Definition.Name, deployDNS.Definition.Namespace)

			err = deployDNS.WaitUntilCondition("Available", 10*time.Minute)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Deployment %q in %q namespace hasn't reached Available state",
					deployDNS.Definition.Name, deployDNS.Definition.Namespace))

			for _, _pod := range appPods {
				By("Running DNS resolution from within a pod")

				cmdDig := fmt.Sprintf("dig +short -t A %s", SPKConfig.WorkloadTestURL)

				Eventually(func(cmd string, mpod *pod.Builder) bool {
					glog.V(spkparams.SPKLogLevel).Infof("Running %q from within a pod %s", cmd, mpod.Definition.Name)
					output, err := mpod.ExecCommand([]string{"/bin/sh", "-c", cmd}, "network-multitool")

					if err != nil {
						glog.V(spkparams.SPKLogLevel).Infof("Failed to run: %q", cmd)

						return false
					}
					glog.V(spkparams.SPKLogLevel).Infof("Command's Output:\n%v\n", output.String())

					return output.String() != ""
				}).WithContext(ctx).WithArguments(cmdDig, _pod).WithPolling(5*time.Second).WithTimeout(
					5*time.Minute).Should(BeTrue(),
					fmt.Sprintf("Command %q from within a pod failed", cmdDig))

				By("Access URL outside the cluster")

				targetURL := fmt.Sprintf("http://%s:%s", SPKConfig.WorkloadTestURL, SPKConfig.WorkloadTestPort)
				cmd := fmt.Sprintf("curl -Ls --max-time 5 -o /dev/null -w '%%{http_code}' %s", targetURL)

				Eventually(func(cmd string, mpod *pod.Builder) bool {

					glog.V(spkparams.SPKLogLevel).Infof("Running %q from within a pod %s", cmd, mpod.Definition.Name)
					output, err := mpod.ExecCommand([]string{"/bin/sh", "-c", cmd}, "network-multitool")

					if err != nil {
						glog.V(spkparams.SPKLogLevel).Infof("Error running %q: %v", cmd, err)

						return false
					}

					glog.V(spkparams.SPKLogLevel).Infof("Command's Output:\n%v\n", output.String())

					normalizedOut := strings.TrimSpace(strings.Trim(output.String(), "'"))

					return normalizedOut == "200" || normalizedOut == "404"
				}).WithContext(ctx).WithArguments(cmd, _pod).WithPolling(5*time.Second).WithTimeout(5*time.Minute).Should(
					BeTrue(), "Failed to access external resource via SPK")

			}

			By(fmt.Sprintf("Scaling down APP deployment %q in %q namespace",
				SPKConfig.WorkloadDCIDeploymentName, SPKConfig.Namespace))

			Eventually(func() bool {

				glog.V(spkparams.SPKLogLevel).Infof("Pulling deployment %q in %q namespace",
					deployApp.Definition.Name, deployApp.Definition.Namespace)
				deployApp, err = deployment.Pull(APIClient, SPKConfig.WorkloadDCIDeploymentName, SPKConfig.Namespace)

				if err != nil {
					return false
				}

				glog.V(spkparams.SPKLogLevel).Infof("Setting replica count for  deployment %q in %q namespace",
					deployApp.Definition.Name, deployApp.Definition.Namespace)
				deployApp = deployApp.WithReplicas(0)

				glog.V(spkparams.SPKLogLevel).Infof("Updating deployment %q in %q namespace",
					deployApp.Definition.Name, deployApp.Definition.Namespace)
				deployApp, err = deployApp.Update()

				if err != nil {
					glog.V(spkparams.SPKLogLevel).Infof("Failed to update deployment %q in %q namespace: %v",
						deployApp.Definition.Name, deployApp.Definition.Namespace, err)

					return false
				}

				glog.V(spkparams.SPKLogLevel).Infof("Successfully updated deployment %q in %q namespace",
					deployApp.Definition.Name, deployApp.Definition.Namespace)

				return true
			}).WithContext(ctx).WithPolling(3*time.Second).WithTimeout(3*time.Minute).Should(BeTrue(),
				fmt.Sprintf("Failed to scale down %q in %q namespace", deployApp.Definition.Name, deployApp.Definition.Namespace))

			glog.V(spkparams.SPKLogLevel).Infof("Wait for pods to terminate")

			Eventually(func() bool {
				appPods, err := pod.List(APIClient, SPKConfig.Namespace,
					metav1.ListOptions{LabelSelector: "app=mttool"})

				if err != nil {
					return false
				}

				return len(appPods) == 0

			}).WithContext(ctx).WithPolling(3*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
				fmt.Sprintf("Pods from deployment %q still exist in %q namespace",
					deployApp.Definition.Name, deployApp.Definition.Namespace))

			By(fmt.Sprintf("Scaling up APP deployment %q in %q namespace",
				SPKConfig.WorkloadDCIDeploymentName, SPKConfig.Namespace))

			Eventually(func() bool {

				glog.V(spkparams.SPKLogLevel).Infof("Pulling deployment %q in %q namespace",
					deployApp.Definition.Name, deployApp.Definition.Namespace)
				deployApp, err = deployment.Pull(APIClient, SPKConfig.WorkloadDCIDeploymentName, SPKConfig.Namespace)

				if err != nil {
					return false
				}

				glog.V(spkparams.SPKLogLevel).Infof("Setting replica count for  deployment %q in %q namespace",
					deployApp.Definition.Name, deployApp.Definition.Namespace)
				deployApp = deployApp.WithReplicas(2)

				glog.V(spkparams.SPKLogLevel).Infof("Updating deployment %q in %q namespace",
					deployApp.Definition.Name, deployApp.Definition.Namespace)
				deployApp, err = deployApp.Update()

				if err != nil {
					glog.V(spkparams.SPKLogLevel).Infof("Failed to update deployment %q in %q namespace: %v",
						deployApp.Definition.Name, deployApp.Definition.Namespace, err)

					return false
				}

				glog.V(spkparams.SPKLogLevel).Infof("Successfully updated deployment %q in %q namespace",
					deployApp.Definition.Name, deployApp.Definition.Namespace)

				return true
			}).WithContext(ctx).WithPolling(3*time.Second).WithTimeout(3*time.Minute).Should(BeTrue(),
				fmt.Sprintf("Failed to scale up %q in %q namespace", deployApp.Definition.Name, deployApp.Definition.Namespace))

			err = deployApp.WaitUntilCondition("Available", 3*time.Minute)
			Expect(err).ToNot(HaveOccurred(),
				fmt.Sprintf("Deployment %q in %q namespace hasn't reached Available state",
					deployApp.Definition.Name, deployApp.Definition.Namespace))

			By("Finding pod backed by deployment")
			Eventually(func() bool {
				appPods, err := pod.List(APIClient, SPKConfig.Namespace,
					metav1.ListOptions{LabelSelector: "app=mttool"})

				if err != nil {
					return false
				}

				return len(appPods) == 2

			}).WithContext(ctx).WithPolling(3*time.Second).WithTimeout(5*time.Minute).Should(BeTrue(),
				"Not enough application pods found")

			appPods, err = pod.List(APIClient, SPKConfig.Namespace,
				metav1.ListOptions{LabelSelector: "app=mttool"})
			Expect(err).ToNot(HaveOccurred(), "Failed to get application pods")

			By("Asserting DNS/NAT46 from new set of application pods")

			for _, _pod := range appPods {
				By("Running DNS resolution from within a pod")

				cmdDig := fmt.Sprintf("dig +short -t A %s", SPKConfig.WorkloadTestURL)

				Eventually(func(cmd string, mpod *pod.Builder) bool {
					glog.V(spkparams.SPKLogLevel).Infof("Running %q from within a pod %s", cmd, mpod.Definition.Name)
					output, err := mpod.ExecCommand([]string{"/bin/sh", "-c", cmd}, "network-multitool")

					if err != nil {
						glog.V(spkparams.SPKLogLevel).Infof("Failed to run: %q", cmd)

						return false
					}
					glog.V(spkparams.SPKLogLevel).Infof("Command's Output:\n%v\n", output.String())

					return output.String() != ""
				}).WithContext(ctx).WithArguments(cmdDig, _pod).WithPolling(5*time.Second).WithTimeout(
					5*time.Minute).Should(BeTrue(),
					fmt.Sprintf("Command %q from within a pod failed", cmdDig))

				By("Access URL outside the cluster")

				targetURL := fmt.Sprintf("http://%s:%s", SPKConfig.WorkloadTestURL, SPKConfig.WorkloadTestPort)
				cmd := fmt.Sprintf("curl -Ls --max-time 5 -o /dev/null -w '%%{http_code}' %s", targetURL)

				Eventually(func(cmd string, mpod *pod.Builder) bool {

					glog.V(spkparams.SPKLogLevel).Infof("Running %q from within a pod %s", cmd, mpod.Definition.Name)
					output, err := mpod.ExecCommand([]string{"/bin/sh", "-c", cmd}, "network-multitool")

					if err != nil {
						glog.V(spkparams.SPKLogLevel).Infof("Error running %q: %v", cmd, err)

						return false
					}

					glog.V(spkparams.SPKLogLevel).Infof("Command's Output:\n%v\n", output.String())

					normalizedOut := strings.TrimSpace(strings.Trim(output.String(), "'"))

					return normalizedOut == "200" || normalizedOut == "404"
				}).WithContext(ctx).WithArguments(cmd, _pod).WithPolling(5*time.Second).WithTimeout(5*time.Minute).Should(
					BeTrue(), "Failed to access external resource via SPK")

			}

			for _, _url := range []string{SPKConfig.IngressTCPIPv4URL, SPKConfig.IngressTCPIPv6URL} {

				if _url != "" {

					glog.V(spkparams.SPKLogLevel).Infof("Trying to reach endpoint %q via F5SPKIngressTCP", _url)

					By("Checking workload via SPK F5SPKIngressTCP")
					Eventually(func(furl string) int {
						result, statusCode, err := url.Fetch(furl, "GET")
						if err != nil {
							glog.V(spkparams.SPKLogLevel).Infof("Failed to reach endpoint %q due to %v",
								furl, err)

							return 1
						}

						glog.V(spkparams.SPKLogLevel).Infof("Successfully reached endpoint %q. Result:\n%v",
							furl, result)

						return statusCode

					}).WithContext(ctx).WithPolling(3*time.Second).WithTimeout(3*time.Minute).WithArguments(_url).Should(
						BeElementOf(200, 404), fmt.Sprintf("Failed to reach workload %q via F5SPKIngressTCP", _url))
				}
			}

		})
	})
