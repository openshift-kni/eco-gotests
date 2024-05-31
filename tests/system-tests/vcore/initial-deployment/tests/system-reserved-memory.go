package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/openshift-kni/eco-goinfra/pkg/nodes"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/vcore/internal/vcoreparams"
)

var _ = Describe(
	"Verify system reserved memory",
	Ordered,
	ContinueOnFailure,
	Label(vcoreparams.Label), func() {
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
		It("Verify system reserved memory for masters", reportxml.ID("60045"),
			Label(vcoreparams.LabelVCoreDeployment), func() {
				kubeletConfigName := "set-sysreserved-master"
				systemReservedBuilder := mco.NewKubeletConfigBuilder(APIClient, kubeletConfigName).
					WithMCPoolSelector("pools.operator.machineconfiguration.openshift.io/master", "").
					WithSystemReserved(vcoreparams.SystemReservedCPU, vcoreparams.SystemReservedMemory)

				if !systemReservedBuilder.Exists() {

					glog.V(100).Infof("Create system-reserved configuration")
					systemReserved, err := systemReservedBuilder.Create()
					Expect(err).ToNot(HaveOccurred(), "Failed to create %s kubeletConfig objects "+
						"with system-reserved definition", kubeletConfigName)

					_, err = nodes.WaitForAllNodesToReboot(
						APIClient,
						20*time.Minute,
						VCoreConfig.ControlPlaneLabelListOption)
					Expect(err).ToNot(HaveOccurred(), "Nodes failed to reboot after applying %s config; %s",
						kubeletConfigName, err)

					Expect(systemReserved.Exists()).To(Equal(true),
						"Failed to setup master system reserved memory, %s kubeletConfig not found; %s",
						kubeletConfigName, err)
				}
			})
	})
