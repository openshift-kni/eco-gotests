package ran_du_system_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/platform"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/stability"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ran-du/internal/randuinittools"
)

var _ = Describe(
	"StabilityNoWorkload",
	Ordered,
	ContinueOnFailure,
	Label("StabilityNoWorkload"), func() {
		var (
			clusterName string
			err         error
		)
		BeforeAll(func() {

			By("Fetching Cluster name")
			clusterName, err = platform.GetOCPClusterName(APIClient)
			Expect(err).ToNot(HaveOccurred(), "Failed to get cluster name")

		})
		It("StabilityNoWorkload", reportxml.ID("74522"), Label("StabilityNoWorkload"), func() {

			outputDir := RanDuTestConfig.StabilityOutputPath
			policiesOutputFile := fmt.Sprintf("%s/stability_policies.log", outputDir)
			ptpOutputFile := fmt.Sprintf("%s/stability_ptp.log", outputDir)
			namespaces := []string{"openshift-etcd", "openshift-apiserver"}

			totalDuration := time.Duration(RanDuTestConfig.StabilityDurationMins) * time.Minute
			interval := time.Duration(RanDuTestConfig.StabilityIntervalMins) * time.Minute
			startTime := time.Now()

			By("Start collecting metrics during the stability test duration defined")
			for time.Since(startTime) < totalDuration {

				if RanDuTestConfig.PtpEnabled {
					err = stability.SavePTPStatus(APIClient, ptpOutputFile, interval)
					if err != nil {
						fmt.Printf("Error, could not save PTP")
					}
				}

				err = stability.SavePolicyStatus(APIClient, clusterName, policiesOutputFile)
				if err != nil {
					fmt.Printf("Error, could not save policies status")
				}

				for _, namespace := range namespaces {
					err = stability.SavePodsRestartsInNamespace(APIClient,
						namespace, fmt.Sprintf("%s/stability_%s.log", outputDir, namespace))
					if err != nil {
						fmt.Printf("Error, could not save Pod restarts")
					}

				}

				time.Sleep(interval)
			}

			// Final check of all values
			By("Check all results")
			var stabilityErrors []string

			// Verify policies
			By("Check Policy changes")
			_, err := stability.VerifyStabilityStatusChange(policiesOutputFile)
			if err != nil {
				stabilityErrors = append(stabilityErrors, err.Error())
			}

			// Verify podRestarts
			By("Check Pod restarts")
			for _, namespace := range namespaces {
				_, err := stability.VerifyStabilityStatusChange(fmt.Sprintf("%s/stability_%s.log", outputDir, namespace))
				if err != nil {
					stabilityErrors = append(stabilityErrors, err.Error())
				}
			}

			// Verify PTP output
			By("Check PTP results")
			if RanDuTestConfig.PtpEnabled {
				_, err = stability.VerifyStabilityStatusChange(ptpOutputFile)
				if err != nil {
					stabilityErrors = append(stabilityErrors, err.Error())
				}
			}

			By("Check if there been any error")
			if len(stabilityErrors) > 0 {
				Expect(stabilityErrors).ToNot(HaveOccurred(), "One or more errors in stability tests:%s", stabilityErrors)
			}

		})
		AfterAll(func() {
		})
	})
