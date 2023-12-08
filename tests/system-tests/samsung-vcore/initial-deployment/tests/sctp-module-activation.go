package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/ocpcli"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/clusteroperator"
	"github.com/openshift-kni/eco-goinfra/pkg/mco"
	"github.com/openshift-kni/eco-goinfra/pkg/nodes"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/samsung-vcore/internal/samsunginittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/samsung-vcore/internal/samsungparams"
)

var _ = Describe(
	"Verify sctp module activation",
	Ordered,
	ContinueOnFailure,
	Label(samsungparams.Label), func() {
		It("Verify sctp module activation", polarion.ID("60086"),
			Label("samsungvcoredeployment"), func() {

				By("Get available samsung-cnf nodes")
				nodesList, err := nodes.List(APIClient, SamsungConfig.SamsungCnfLabelListOption)
				Expect(err).ToNot(HaveOccurred(), "Failed to get samsung-cnf nodes list; %s", err)
				Expect(len(nodesList)).ToNot(Equal(0), "samsung-cnf nodes list is empty")

				sctpBuilder := mco.NewMCBuilder(APIClient, samsungparams.SctpModuleName)
				if !sctpBuilder.Exists() {

					By("Apply sctp config using shell method")

					sctpModuleTemplateName := "sctp-module.yaml"
					varsToReplace := make(map[string]interface{})
					varsToReplace["SctpModuleName"] = "load-sctp-module"
					varsToReplace["McNodeRole"] = samsungparams.CnfMCSelector
					homeDir, err := os.UserHomeDir()
					Expect(err).ToNot(HaveOccurred(), "user home directory not found; %s", err)

					destinationDirectoryPath := filepath.Join(homeDir, samsungparams.ConfigurationFolderName)

					workingDir, err := os.Getwd()
					Expect(err).ToNot(HaveOccurred(), err)
					templateDir := filepath.Join(workingDir, samsungparams.TemplateFilesFolder)

					err = ocpcli.ApplyConfigFile(
						templateDir,
						sctpModuleTemplateName,
						destinationDirectoryPath,
						sctpModuleTemplateName,
						varsToReplace)
					Expect(err).To(BeNil(), fmt.Sprint(err))

					Expect(sctpBuilder.Exists()).To(Equal(true),
						"Failed to create %s CRD", sctpModuleTemplateName)

					_, err = nodes.WaitForAllNodesToReboot(
						APIClient,
						20*time.Minute,
						SamsungConfig.SamsungCnfLabelListOption)
					Expect(err).ToNot(HaveOccurred(), "Nodes failed to reboot after applying %s config; %s",
						sctpModuleTemplateName, err)

				}

				_, err = clusteroperator.WaitForAllClusteroperatorsAvailable(APIClient, 60*time.Second)
				Expect(err).ToNot(HaveOccurred(), "Error waiting for all available clusteroperators: %s", err)

				glog.V(100).Infof("Verify SCTP was activated on each %s node", SamsungConfig.SamsungCnfLabel)

				for _, node := range nodesList {
					checkCmd := "lsmod | grep sctp"

					output, err := ocpcli.ExecuteViaDebugPodOnNode(node.Object.Name, checkCmd)
					Expect(err).ToNot(HaveOccurred(), "Failed to execute command on node %s; %s",
						node.Object.Name, err)
					Expect(output).To(ContainSubstring("sctp"),
						"Failed to enable SCTP on %s node: %s", node.Object.Name, output)
				}

			})
	})
