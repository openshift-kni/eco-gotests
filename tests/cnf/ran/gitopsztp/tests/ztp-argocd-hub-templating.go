package tests

import (
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/secret"
	"github.com/openshift-kni/eco-goinfra/pkg/sriov"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/gitopsztp/internal/gitdetails"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/gitopsztp/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/gitopsztp/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranhelper"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	corev1 "k8s.io/api/core/v1"
	policiesv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
)

var _ = Describe("ZTP Argo CD Hub Templating Tests", Label(tsparams.LabelArgoCdHubTemplatingTestCases), func() {
	BeforeEach(func() {
		By("checking the ZTP version")
		versionInRange, err := ranhelper.IsVersionStringInRange(RANConfig.ZTPVersion, "4.12", "")
		Expect(err).ToNot(HaveOccurred(), "Failed to check if ZTP version is in range")

		if !versionInRange {
			Skip("This test requires a ZTP version of at least 4.12")
		}

		By("ensuring the test namespace exists")
		_, err = namespace.NewBuilder(HubAPIClient, tsparams.TestNamespace).Create()
		Expect(err).ToNot(HaveOccurred(), "Failed to create the test namespace")
	})

	AfterEach(func() {
		By("resetting the policies app back to the original settings")
		err := gitdetails.SetGitDetailsInArgoCd(
			tsparams.ArgoCdPoliciesAppName, tsparams.ArgoCdAppDetails[tsparams.ArgoCdPoliciesAppName], true, false)
		Expect(err).ToNot(HaveOccurred(), "Failed to reset the git details for the policies app")

		By("removing the hub templating leftovers if any exist")
		network, err := sriov.PullNetwork(Spoke1APIClient, tsparams.TestNamespace, RANConfig.SriovOperatorNamespace)
		if err == nil {
			err = network.DeleteAndWait(tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to delete SR-IOV network")
		}

		By("removing the CGU if it exists")
		cguBuilder, err := cgu.Pull(
			HubAPIClient, tsparams.HubTemplatingCguName, tsparams.HubTemplatingCguNamespace)
		if err == nil {
			_, err = cguBuilder.DeleteAndWait(5 * time.Minute)
			Expect(err).ToNot(HaveOccurred(), "Failed to delete and wait for CGU to be deleted")
		}
	})

	// 54240 - Hub-side ACM templating with TALM
	It("should report an error for using autoindent function where not allowed", reportxml.ID("54240"), func() {
		setupHubTemplateTest(tsparams.ZtpTestPathTemplatingAutoIndent)

		By("validating TALM reported a policy error")
		assertTalmPodLog(HubAPIClient, "policy has hub template error")

		By("validating the specific error using the policy message")
		policy, err := ocm.PullPolicy(
			HubAPIClient, tsparams.TestNamespace+"."+tsparams.HubTemplatingPolicyName, RANConfig.Spoke1Name)
		Expect(err).ToNot(HaveOccurred(), "Failed to pull hub side templating policy")

		_, err = policy.WaitForStatusMessageToContain(
			"wrong type for value; expected string; got int", tsparams.ArgoCdChangeTimeout)
		Expect(err).ToNot(HaveOccurred(), "Failed to validate error using policy message")
	})

	When("supported ACM hub side templating is used", func() {
		// 54240 - Hub-side ACM templating with TALM
		It("should create the policy successfully with a valid template", reportxml.ID("54240"), func() {
			By("checking the ZTP version")
			versionInRange, err := ranhelper.IsVersionStringInRange(RANConfig.ZTPVersion, "4.16", "")
			Expect(err).ToNot(HaveOccurred(), "Failed to check if ZTP version is in range")

			validTestPath := tsparams.ZtpTestPathTemplatingValid
			if versionInRange {
				validTestPath = tsparams.ZtpTestPathTemplatingValid416
			}

			// We must create the secret before the test since creating it by ZTP is not allowed.
			validSecret, err := secret.NewBuilder(
				HubAPIClient, tsparams.HubTemplatingSecretName, tsparams.TestNamespace, corev1.SecretTypeOpaque).
				WithData(map[string][]byte{"vlanQoS": []byte("MAo=")}).
				Create()
			Expect(err).ToNot(HaveOccurred(), "Failed to create hub templating secret")

			DeferCleanup(func() {
				err := validSecret.Delete()
				Expect(err).ToNot(HaveOccurred(), "Failed to clean up hub templating secret")
			})

			setupHubTemplateTest(validTestPath)

			By("validating the policy reaches compliant status")
			policy, err := ocm.PullPolicy(HubAPIClient, tsparams.HubTemplatingPolicyName, tsparams.TestNamespace)
			Expect(err).ToNot(HaveOccurred(), "Failed to get policy from hub cluster")

			err = policy.WaitUntilComplianceState(policiesv1.Compliant, tsparams.ArgoCdChangeTimeout)
			Expect(err).ToNot(HaveOccurred(), "Failed to wait for policy to become compliant")
		})
	})
})

// setupHubTemplateTest extracts the core setup logic for the hub templating test cases.
func setupHubTemplateTest(ztpTestPath string) {
	By("updating the Argo CD git path")

	exists, err := gitdetails.UpdateArgoCdAppGitPath(tsparams.ArgoCdPoliciesAppName, ztpTestPath, true)
	if !exists {
		Skip(err.Error())
	}

	Expect(err).ToNot(HaveOccurred(), "Failed to update Argo CD git path")

	By("waiting for the policy to exist")

	_, err = helper.WaitForPolicyToExist(
		HubAPIClient, tsparams.HubTemplatingPolicyName, tsparams.TestNamespace, tsparams.ArgoCdChangeTimeout)
	Expect(err).ToNot(HaveOccurred(), "Failed to wait for hub templating policy to be created")

	By("creating the CGU")

	cguBuilder := cgu.NewCguBuilder(
		HubAPIClient, tsparams.HubTemplatingCguName, tsparams.HubTemplatingCguNamespace, 1).
		WithCluster(RANConfig.Spoke1Name).
		WithManagedPolicy(tsparams.HubTemplatingPolicyName)
	cguBuilder.Definition.Spec.RemediationStrategy.Timeout = 10

	_, err = cguBuilder.Create()
	Expect(err).ToNot(HaveOccurred(), "Failed to create hub templating CGU")
}

// assertTalmPodLog asserts that the TALM pod log contains the expected substring.
func assertTalmPodLog(client *clients.Settings, expectedSubstring string) {
	glog.V(tsparams.LogLevel).Infof("Waiting for TALM log to report: '%s'", expectedSubstring)

	Eventually(func() string {
		podList, err := pod.List(client, ranparam.OpenshiftOperatorNamespace)
		Expect(err).ToNot(HaveOccurred(), "Failed to list pods is openshift operator namespace")
		Expect(podList).ToNot(BeEmpty(), "Failed to find any pods in the openshift operator namespace")

		var podLog string

		for _, podBuilder := range podList {
			if strings.HasPrefix(podBuilder.Object.Name, tsparams.TalmHubPodName) {
				glog.V(tsparams.LogLevel).Infof("Checking logs for pod %s", podBuilder.Object.Name)

				podLog, err = podBuilder.GetLog(1*time.Minute, ranparam.TalmContainerName)
				Expect(err).ToNot(HaveOccurred(), "Failed to get TALM pod log")

				break
			}
		}

		return podLog
	}, tsparams.ArgoCdChangeTimeout, tsparams.ArgoCdChangeInterval).
		Should(ContainSubstring(expectedSubstring), "Failed to assert TALM pod log contains %s", expectedSubstring)
}
