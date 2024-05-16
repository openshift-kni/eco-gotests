package vcorecommon

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/files"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/shell"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

// VerifyHelmDeploymentProcedure asserts Helm deployment procedure.
func VerifyHelmDeploymentProcedure(ctx SpecContext) {
	glog.V(vcoreparams.VCoreLogLevel).Infof("Verify Helm could be installed and works correctly")

	homeDir, err := os.UserHomeDir()
	Expect(err).To(BeNil(), fmt.Sprint(err))

	vcoreConfigsFolder := filepath.Join(homeDir, vcoreparams.ConfigurationFolderName)

	helmScriptURL := "https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3"
	helmScriptName := "get_helm.sh"
	helmScriptLocalPath := filepath.Join(vcoreConfigsFolder, helmScriptName)

	glog.V(100).Infof("Download %s script", helmScriptName)
	err = files.DownloadFile(helmScriptURL, helmScriptName, vcoreConfigsFolder)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to download %s file locally from the %s due to %v",
		helmScriptName, helmScriptURL, err))

	glog.V(100).Infof("Make %s script executable", helmScriptName)

	chmodCmd := fmt.Sprintf("chmod 700 %s", helmScriptLocalPath)
	_, err = shell.ExecuteCmd(chmodCmd)
	Expect(err).ToNot(HaveOccurred(), "failed to make %s script executable due to %w",
		helmScriptLocalPath, err)

	glog.V(100).Info("Install Helm")

	os.Setenv("VERIFY_CHECKSUM", "false")

	_, err = shell.ExecuteCmd(helmScriptLocalPath)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to execute %s script due to %v",
		helmScriptLocalPath, err))

	glog.V(100).Info("Check HELM working properly")

	cmd := "helm version"
	result, err := shell.ExecuteCmd(cmd)
	Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("failed to check helm version due %v", err))
	Expect(strings.Contains(string(result), "version.BuildInfo")).To(Equal(true),
		fmt.Sprintf("Helm was not installed properly; %v", string(result)))
} // func VerifyHelmDeploymentProcedure (ctx SpecContext)

// VerifyHelmSuite container that contains tests for the Helm verification.
func VerifyHelmSuite() {
	Describe(
		"Helm validation",
		Label(vcoreparams.LabelVCoreOperators), func() {
			BeforeAll(func() {
				By(fmt.Sprintf("Asserting %s folder exists", vcoreparams.ConfigurationFolderName))

				homeDir, err := os.UserHomeDir()
				Expect(err).To(BeNil(), fmt.Sprint(err))

				vcoreConfigsFolder := filepath.Join(homeDir, vcoreparams.ConfigurationFolderName)

				glog.V(100).Infof("vcoreConfigsFolder: %s", vcoreConfigsFolder)

				if err := os.Mkdir(vcoreConfigsFolder, 0755); os.IsExist(err) {
					glog.V(100).Infof("%s folder already exists", vcoreConfigsFolder)
				}
			})

			It("Verify Helm deployment procedure",
				Label("helm"), reportxml.ID("60085"), VerifyHelmDeploymentProcedure)
		})
}
