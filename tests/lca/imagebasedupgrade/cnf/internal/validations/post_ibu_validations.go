package cnfibuvalidations

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/deployment"
	"github.com/openshift-kni/eco-goinfra/pkg/lca"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-goinfra/pkg/statefulset"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/lca/internal/seedimage"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfclusterinfo"
	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfinittools"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfparams"
)

var (
	ibu      *lca.ImageBasedUpgradeBuilder
	seedInfo *seedimage.SeedImageContent
	err      error
)

// PostUpgradeValidations is a dedicated func to run post upgrade test validations.
func PostUpgradeValidations() {
	Describe(
		"PostIBUValidations",
		Ordered,
		Label("PostIBUValidations"), func() {
			BeforeAll(func() {
				By("Retrieve seed image info", func() {
					Eventually(func() bool {
						ibu, err = lca.PullImageBasedUpgrade(TargetSNOAPIClient)
						if err != nil {
							return false
						}

						seedImage, err := seedimage.GetContent(TargetSNOAPIClient, ibu.Definition.Spec.SeedImageRef.Image)
						if err != nil {
							return false
						}

						seedInfo = seedImage

						return true
					}, 2*time.Minute, 5*time.Second).Should(BeTrue(), "Failed to get seed image info")
				})
			})

			ValidateClusterVersionCSVs()

			ValidateClusterID()

			ValidatePodsSeedName()

			ValidateReboots()

			ValidateOperatorRollouts()

			ValidateClusterHostname()

			ValidateNetworkConfig()

			ValidateSeedHostnameRefLogs()

			ValidateSeedHostnameRefEtcd()

			ValidateDUConfig()

			ValidateWorkload()

			ValidateWorkloadPV()

			ValidateNoImagesPulled()

			ValidateSeedRefMetrics()

			ValidateSeedRefLogs()
		})
}

// ValidateClusterVersionCSVs check clusterversion and operator CSVs version after upgrade.
func ValidateClusterVersionCSVs() {
	It("Validate Cluster version and operators CSVs", reportxml.ID("71387"),
		Label("ValidateClusterVersionOperatorCSVs"), func() {
			By("Validate upgraded cluster version reports correct version", func() {
				Expect(cnfclusterinfo.PreUpgradeClusterInfo.Version).
					ToNot(Equal(cnfclusterinfo.PostUpgradeClusterInfo.Version), "Cluster version hasn't changed")
			})
			By("Validate CSVs were upgraded", func() {
				for _, preUpgradeOperator := range cnfclusterinfo.PreUpgradeClusterInfo.Operators {
					for _, postUpgradeOperator := range cnfclusterinfo.PostUpgradeClusterInfo.Operators {
						Expect(preUpgradeOperator).ToNot(Equal(postUpgradeOperator), "Operator %s was not upgraded", preUpgradeOperator)
					}
				}
			})
		})
}

// ValidateClusterID check if cluster ID remains the same after upgrade.
func ValidateClusterID() {
	It("Validate Cluster ID", reportxml.ID("71388"), Label("ValidateClusterID"), func() {
		By("Validate cluster ID remains the same", func() {
			Expect(cnfclusterinfo.PreUpgradeClusterInfo.ID).
				To(Equal(cnfclusterinfo.PostUpgradeClusterInfo.ID), "Cluster ID has changed")
		})
	})
}

// ValidatePodsSeedName check if no pods are using seed's name after upgrade.
func ValidatePodsSeedName() {
	It("Validate no pods using seed name", reportxml.ID("71389"), Label("ValidatePodsSeedName"), func() {
		By("Validate no pods are using seed's name", func() {
			podList, err := pod.ListInAllNamespaces(TargetSNOAPIClient, v1.ListOptions{})
			Expect(err).ToNot(HaveOccurred(), "Failed to list pods")

			for _, pod := range podList {
				Expect(pod.Object.Name).ToNot(ContainSubstring(seedInfo.SNOHostname),
					"Pod %s is using seed's name", pod.Object.Name)
			}
		})
	})
}

// ValidateReboots check if no extra reboots are performed after upgrade.
func ValidateReboots() {
	It("Validate no extra reboots after upgrade", reportxml.ID("72962"), Label("ValidateUpgradeReboots"), func() {
		By("Validate no extra reboots after upgrade", func() {
			rebootCmd := "last | grep reboot | wc -l"
			rebootCount, err := cluster.ExecCmdWithStdout(TargetSNOAPIClient, rebootCmd)
			Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)

			for _, stdout := range rebootCount {
				Expect(strings.ReplaceAll(stdout, "\n", "")).To(Equal("1"), "Extra reboots detected: %s", stdout)
			}
		})
	})
}

// ValidateOperatorRollouts check if no operator rollouts are performed after upgrade.
func ValidateOperatorRollouts() {
	It("Validate no operator rollouts post upgrade", reportxml.ID("73048"), Label("ValidateOperatorRollouts"), func() {
		By("Validate no cluster operator rollouts after upgrade", func() {
			rolloutCmd := "grep -Ri RevisionTrigger /var/log/pods/ | wc -l"
			rolloutCheck, err := cluster.ExecCmdWithStdout(TargetSNOAPIClient, rolloutCmd)
			Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)

			for _, stdout := range rolloutCheck {
				intRollouts, err := strconv.Atoi(strings.ReplaceAll(stdout, "\n", ""))
				Expect(err).ToNot(HaveOccurred(), "could not convert string to int: %s", err)
				Expect(intRollouts).ToNot(BeNumerically(">", 0), "Cluster operator rollouts detected: %s", stdout)
			}
		})
	})
}

// ValidateClusterHostname check if cluster hostname is updated after upgrade.
func ValidateClusterHostname() {
	It("Validate upgraded cluster hostname", reportxml.ID("71390"), Label("ValidateHostName"), func() {
		By("Validate upgraded cluster hostname", func() {
			hostnameCmd := "cat /etc/hostname"
			hostnameRes, err := cluster.ExecCmdWithStdout(TargetSNOAPIClient, hostnameCmd)
			Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)

			for _, stdout := range hostnameRes {
				Expect(stdout).ToNot(ContainSubstring(seedInfo.SNOHostname),
					"Target hostname %s is using seed's hostname", stdout)
			}
		})
	})
}

// ValidateNetworkConfig check if network configuration is updated after upgrade.
func ValidateNetworkConfig() {
	It("Validate upgraded cluster network configuration", reportxml.ID("71391"), Label("ValidateNetworkConfig"), func() {
		By("Validate upgraded cluster network configuration", func() {
			var recFound bool

			digCmd := "dig -q A `cat /etc/hostname` AAAA `cat /etc/hostname` +short"
			digRes, err := cluster.ExecCmdWithStdout(TargetSNOAPIClient, digCmd)
			Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)

			ipCmd := "/usr/sbin/ip --brief a s dev br-ex"
			ipRes, err := cluster.ExecCmdWithStdout(TargetSNOAPIClient, ipCmd)
			Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)

			for _, dnsRec := range digRes {
				for _, ipOut := range ipRes {
					for _, rec := range strings.Split(dnsRec, "\n") {
						if rec != "" {
							if strings.Contains(ipOut, rec) {
								glog.V(100).Infof("Found DNS record %s in network configuration", rec)

								recFound = true
							}
						}
					}
				}
			}

			Expect(recFound).To(BeTrue(), "Could not find any of the node hostname DNS records in the network configuration")
		})
	})
}

// ValidateSeedHostnameRefLogs check if no seed hostname references are present in pod logs.
func ValidateSeedHostnameRefLogs() {
	It("Validate no seed hostname references in pod logs", reportxml.ID("71392"), Label("ValidateSeedRefPodLogs"), func() {
		By("Validate no seed hostname references in pod logs", func() {
			logCmd := "grep -Ri " + seedInfo.SNOHostname + " /var/log/pods | grep -v lifecycle-agent-controller-manager | wc -l"
			logRes, err := cluster.ExecCmdWithStdout(TargetSNOAPIClient, logCmd)
			Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)

			for _, stdout := range logRes {
				seedLogRef, err := strconv.Atoi(strings.ReplaceAll(stdout, "\n", ""))
				Expect(err).ToNot(HaveOccurred(), "could not convert string to int: %s", err)
				Expect(seedLogRef).ToNot(BeNumerically(">", 0), "Seed hostname references detected in pod logs: %s", stdout)
			}
		})
	})
}

// ValidateSeedHostnameRefEtcd check if no seed hostname references are present in etcd.
func ValidateSeedHostnameRefEtcd() {
	It("Validate no seed hostname references in etcd", reportxml.ID("71393"), Label("ValidateSeedRefEtcd"), func() {
		By("Validate no seed hostname references in ectd", func() {
			etcdPod, err := pod.ListByNamePattern(TargetSNOAPIClient, "etcd", "openshift-etcd")
			Expect(err).ToNot(HaveOccurred(), "Failed to get etcd pod")

			etcdCmd := fmt.Sprintf(
				"for key in $(etcdctl get --prefix / --keys-only | grep -v ^$); do"+
					"  value=$(etcdctl get --print-value-only $key | tr -d '\\0'); "+
					"  if [[ $value == *%s* ]]; then"+
					"    echo Key: $key contains seed reference;"+
					"    break; "+
					"  fi;"+
					"done",
				seedInfo.ClusterName,
			)

			getKeyCmd := []string{"bash", "-c", etcdCmd}
			getKeyOut, err := etcdPod[0].ExecCommand(getKeyCmd, "etcdctl")
			Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)
			Expect(getKeyOut.String()).ToNot(
				ContainSubstring("contains seed reference"),
				"Seed hostname references detected in etcd: %s",
				getKeyOut.String(),
			)
		})
	})
}

// ValidateDUConfig check if pre-upgrade DU configs are present after upgrade.
func ValidateDUConfig() {
	It("Validate pre-upgrade DU configs are present after upgrade", reportxml.ID("71394"),
		Label("ValidateDUConfig"), func() {
			By("Validate pre-upgrade DU configs are present after upgrade", func() {
				By("Validate SR-IOV networks", func() {
					for _, preUpgradeNet := range cnfclusterinfo.PreUpgradeClusterInfo.SriovNetworks {
						glog.V(100).Infof("Checking for SR-IOV network %s", preUpgradeNet)
						Expect(cnfclusterinfo.PostUpgradeClusterInfo.SriovNetworks).
							To(ContainElement(preUpgradeNet), "SR-IOV network %s was not found after upgrade", preUpgradeNet)
					}
				})

				By("Validate SR-IOV policies", func() {
					for _, preUpgradePolicy := range cnfclusterinfo.PreUpgradeClusterInfo.SriovNetworkNodePolicies {
						glog.V(100).Infof("Checking for SR-IOV policy %s", preUpgradePolicy)
						Expect(cnfclusterinfo.PostUpgradeClusterInfo.SriovNetworkNodePolicies).
							To(ContainElement(preUpgradePolicy), "SR-IOV policy %s was not found after upgrade", preUpgradePolicy)
					}
				})
			})
		})
}

// ValidateWorkload check if test workload pods are running without errors.
func ValidateWorkload() {
	It("Validate test workload pods are running without errors", reportxml.ID("71395"), Label("ValidateDUPods"), func() {
		By("Validate test workload pods are running without errors", func() {
			for _, workload := range cnfclusterinfo.PostUpgradeClusterInfo.WorkloadResources {
				workloadNS := workload.Namespace

				for _, deploy := range workload.Objects.Deployment {
					getDeploy, err := deployment.Pull(TargetSNOAPIClient, deploy, workloadNS)
					Expect(err).ToNot(HaveOccurred(),
						"Unable to pull Deployment %s in namespace %s is not ready",
						deploy, workloadNS)

					podReady := getDeploy.IsReady(cnfparams.WorkloadReadyTimeout)
					Expect(podReady).To(BeTrue(), "Deployment %s in namespace %s is not ready", deploy, workloadNS)
				}

				for _, sset := range workload.Objects.StatefulSet {
					getStatefulSet, err := statefulset.Pull(TargetSNOAPIClient, sset, workloadNS)
					Expect(err).ToNot(HaveOccurred(), "Unable to pull StatefulSet %s in namespace %s is not ready", sset, workloadNS)

					ssetReady := getStatefulSet.IsReady(cnfparams.WorkloadReadyTimeout)
					Expect(ssetReady).To(BeTrue(), "StatefulSet %s in namespace %s is not ready", sset, workloadNS)
				}
			}
		})
	})
}

// ValidateWorkloadPV check if pre-upgrade files written to PVs are preserved during upgrade.
func ValidateWorkloadPV() {
	It("Validate pre-upgrade files written to PVs are preserved during upgrade",
		reportxml.ID("71396"),
		Label("ValidatePVFiles"),
		func() {
			By("Validate pre-upgrade files written to PVs are preserved during upgrade", func() {
				Expect(cnfclusterinfo.PreUpgradeClusterInfo.WorkloadPVs.Digest).To(
					BeIdenticalTo(cnfclusterinfo.PostUpgradeClusterInfo.WorkloadPVs.Digest),
					"Files digest do not match %s vs %s",
					cnfclusterinfo.PreUpgradeClusterInfo.WorkloadPVs.Digest,
					cnfclusterinfo.PostUpgradeClusterInfo.WorkloadPVs.Digest,
				)
			})
		})
}

// ValidateNoImagesPulled check if no images are being pulled after upgrade.
func ValidateNoImagesPulled() {
	It("Validate no images are being pulled after upgrade", reportxml.ID("73051"), Label("ValidateImagePull"), func() {
		By("Validate no images are being pulled after upgrade", func() {
			catSrc, err := olm.ListCatalogSources(TargetSNOAPIClient, "openshift-marketplace")
			Expect(err).ToNot(HaveOccurred(), "Failed to list catalog sources")

			excludeImages := "| grep -v "
			for i, catalogSource := range catSrc {
				excludeImages += catalogSource.Definition.Spec.Image
				if i < len(catSrc)-1 {
					excludeImages += " | grep -v "
				}
			}

			logCmd := "journalctl -l --system | grep Pulling | grep image:"
			logRes, err := cluster.ExecCmdWithStdout(TargetSNOAPIClient, logCmd+excludeImages)
			Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)

			for _, stdout := range logRes {
				Expect(len(stdout)).ToNot(BeNumerically(">", 1), "Images are being pulled after upgrade: %s", stdout)
			}
		})
	})
}

// ValidateSeedRefMetrics check if no seed references are present in the exported metrics.
func ValidateSeedRefMetrics() {
	It("Validate no seed references in the exported metrics", reportxml.ID("71397"), Label("ValidateMetrics"), func() {
		By("Validate no seed references in the exported metrics", func() {
			promPod, err := pod.ListByNamePattern(TargetSNOAPIClient, "prometheus-k8s", "openshift-monitoring")
			Expect(err).ToNot(HaveOccurred(), "Failed to get prometheus pod")

			getMetricsCmd := []string{"bash", "-c", "curl -L http://localhost:9090/api/v1/targets"}
			getMetricsOut, err := promPod[0].ExecCommand(getMetricsCmd)
			Expect(err).ToNot(HaveOccurred(), "could not execute command: %s", err)
			Expect(getMetricsOut.String()).ToNot(
				ContainSubstring(seedInfo.SNOHostname),
				"Seed cluster name references detected in prometheus metrics: %s",
				getMetricsOut.String(),
			)
		})
	})
}

// ValidateSeedRefLogs check if no seed references are present in the exported logs.
func ValidateSeedRefLogs() {
	It("Validate no seed references in the exported logs", reportxml.ID("71398"), Label("ValidateLogs"), func() {
		By("Validate no seed references in the exported logs", func() {
			deployContainer := pod.NewContainerBuilder("kcat",
				CNFConfig.IbuKcatImage, []string{"sleep", "86400"})
			deployCfg, err := deployContainer.GetContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "Failed to get kcat container config")

			createDeploy := deployment.NewBuilder(
				TargetSNOAPIClient,
				"kcat",
				"default",
				map[string]string{"kcat": ""},
				*deployCfg,
			)
			_, err = createDeploy.CreateAndWaitUntilReady(2 * time.Minute)
			Expect(err).ToNot(HaveOccurred(), "Failed to create kcat deployment")

			kcatPod, err := pod.ListByNamePattern(TargetSNOAPIClient, "kcat", "default")
			Expect(err).ToNot(HaveOccurred(), "Failed to get kcat pod")

			logCmd := fmt.Sprintf("kcat  -b %s -C -t %s -C -q -o end -c 200", CNFConfig.IbuKcatBroker, CNFConfig.IbuKcatTopic)
			getLogsCmd := []string{"sh", "-c", logCmd}

			var kcatPodOut string

			Eventually(func() bool {
				getLogsOut, err := kcatPod[0].ExecCommand(getLogsCmd, "kcat")
				if err != nil {
					return false
				}

				kcatPodOut = getLogsOut.String()

				return true
			}, 1*time.Minute, 5*time.Second).Should(BeTrue(), "Failed to get kcat logs")

			Expect(kcatPodOut).ToNot(
				ContainSubstring(seedInfo.SNOHostname),
				"Seed cluster name references detected in kcat logs: %s", kcatPodOut)
		})
	})
}
