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
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

		It("Asserts DNS resoulution from new deployment", reportxml.ID("63171"), Label("spkdns46new"), func(ctx SpecContext) {
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

		It("Asserts DNS resolution from existing deployment", reportxml.ID("66091"), Label("spkdns46existing"), func() {

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
	})
