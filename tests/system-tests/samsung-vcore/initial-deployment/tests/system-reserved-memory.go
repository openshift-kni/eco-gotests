package tests

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/samsung-vcore/internal/samsunginittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/samsung-vcore/internal/samsungparams"
)

var _ = Describe(
	"Verify system reserved memory",
	Ordered,
	ContinueOnFailure,
	Label(samsungparams.Label), func() {
		BeforeAll(func() {
			By(fmt.Sprintf("Asserting %s folder exists", samsungparams.ConfigurationFolderName))

			homeDir, err := os.UserHomeDir()
			Expect(err).To(BeNil(), fmt.Sprint(err))

			samsungConfigsFolder := filepath.Join(homeDir, samsungparams.ConfigurationFolderName)

			glog.V(100).Infof("samsungConfigsFolder: %s", samsungConfigsFolder)

			if err := os.Mkdir(samsungConfigsFolder, 0755); os.IsExist(err) {
				glog.V(100).Infof("%s folder already exists", samsungConfigsFolder)
			}
		})
		It("Verify system reserved memory for masters", reportxml.ID("60045"),
			Label("samsungvcoredeployment"), func() {
				kubeletConfigName := "set-sysreserved-master"
				systemReservedBuilder := mco.NewKubeletConfigBuilder(APIClient, kubeletConfigName).
					WithMCPoolSelector("pools.operator.machineconfiguration.openshift.io/master", "").
					WithSystemReserved(samsungparams.SystemReservedCPU, samsungparams.SystemReservedMemory)

				if systemReservedBuilder.Exists() {
					glog.V(100).Infof("Cleanup system-reserved configuration")
					err := systemReservedBuilder.Delete()
					Expect(err).ToNot(HaveOccurred(), "Failed to delete %s kubeletConfig objects "+
						"with system-reserved definition", kubeletConfigName)
				}

				glog.V(100).Infof("Create system-reserved configuration")
				systemReserved, err := systemReservedBuilder.Create()
				Expect(err).ToNot(HaveOccurred(), "Failed to create %s kubeletConfig objects "+
					"with system-reserved definition", kubeletConfigName)

				Expect(systemReserved.Exists()).To(Equal(true),
					"Failed to setup master system reserved memory, %s kubeletConfig not found; %s",
					kubeletConfigName, err)
			})
	})
