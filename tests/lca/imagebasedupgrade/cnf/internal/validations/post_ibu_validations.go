package cnfibuvalidations

import (
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/lca"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfclusterinfo"
	"github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/internal/seedimage"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openshift-kni/eco-gotests/tests/lca/imagebasedupgrade/cnf/internal/cnfinittools"
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
					ibu, err = lca.PullImageBasedUpgrade(TargetSNOAPIClient)
					Expect(err).NotTo(HaveOccurred(), "error pulling ibu resource from cluster")

					seedInfo, err = seedimage.GetContent(TargetSNOAPIClient, ibu.Definition.Spec.SeedImageRef.Image)
					Expect(err).NotTo(HaveOccurred(), "error getting seed image info")
				})
			})

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

			It("Validate Cluster ID", reportxml.ID("71388"), Label("ValidateClusterID"), func() {
				By("Validate cluster ID remains the same", func() {
					Expect(cnfclusterinfo.PreUpgradeClusterInfo.ID).
						To(Equal(cnfclusterinfo.PostUpgradeClusterInfo.ID), "Cluster ID has changed")
				})
			})

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
		})
}
