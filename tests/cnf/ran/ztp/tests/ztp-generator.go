package tests

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/reportxml"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/ztp/internal/tsparams"
	"github.com/openshift-kni/eco-gotests/tests/internal/cluster"
	"gopkg.in/yaml.v2"
)

var _ = Describe("ZTP Generator Tests", Label(tsparams.LabelGeneratorTestCases, ranparam.LabelNoContainer), func() {
	var siteConfigPath string

	BeforeEach(func() {
		By("getting current user")
		output, err := cluster.ExecLocalCommand(time.Minute, "whoami")
		Expect(err).ToNot(HaveOccurred(), "Failed to run get current user")

		user := strings.TrimSpace(output)

		By("checking siteconfig path")
		siteConfigPath = fmt.Sprintf("/home/%s/site-configs", user)
		_, err = os.Stat(siteConfigPath)
		Expect(err).ToNot(HaveOccurred(), "Failed to find site config repo at '%s'", siteConfigPath)
	})

	AfterEach(func() {
		By("deleting the generated manifests and policies")
		_, err := cluster.ExecLocalCommand(time.Minute, "sudo", "rm", "-rf", siteConfigPath+"/siteconfig/out")
		Expect(err).ToNot(HaveOccurred(), "Failed to delete siteconfig output")

		_, err = cluster.ExecLocalCommand(time.Minute, "sudo", "rm", "-rf", siteConfigPath+"/policygentemplates/out")
		Expect(err).ToNot(HaveOccurred(), "Failed to delete policygentemplates output")
	})

	// 54355 - Generation of CRs for a single site from ztp container
	It("generates and installs time crs, manifests, and policies, and verifies they are present",
		reportxml.ID("54355"), func() {
			By("validating the image version for the site generator")
			var ztpImageTag string

			// Since brew is a lot faster than skopeo, we want to use it if its available
			brew, err := cluster.ExecLocalCommand(time.Minute, "which", "brew")

			if err != nil || brew == "" {
				By("using skopeo to find the image tag")
				cmd := fmt.Sprintf(
					"skopeo list-tags docker://%s | grep %s", RANConfig.ZtpSiteGenerateImage, RANConfig.ZTPVersion) +
					" | sort -V | tail -1 | tr -d '\"' | tr -d ','"

				output, err := cluster.ExecLocalCommand(time.Minute, "bash", "-c", cmd)
				Expect(err).ToNot(HaveOccurred(), "Failed to get output from skopeo")

				ztpImageTag = strings.TrimSpace(output)
			} else {
				By("using brew to find the image tag")
				cmd := "brew list-builds --package=ztp-site-generate-container --state=COMPLETE --quiet" +
					fmt.Sprintf(" | grep %s", RANConfig.ZTPVersion) +
					" | sort -V | tail -1 | awk '{ print $1 }' | sed -nr 's/.*-(v.*)$/\\1/p'"

				output, err := cluster.ExecLocalCommand(time.Minute, "bash", "-c", cmd)
				Expect(err).ToNot(HaveOccurred(), "Failed to get output from brew")

				ztpImageTag = strings.TrimSpace(output)
			}

			glog.V(tsparams.LogLevel).Infof("Detected ZTP image tag '%s'", ztpImageTag)

			By("generating the install time CRs and manifests")
			_, err = cluster.ExecLocalCommand(
				time.Minute,
				"podman",
				"run",
				"--rm",
				"-v",
				fmt.Sprintf("%s/siteconfig/:/resources:Z", siteConfigPath),
				fmt.Sprintf("%s:%s", RANConfig.ZtpSiteGenerateImage, ztpImageTag),
				"generator",
				"install",
				"-E",
				"/resources/")
			Expect(err).ToNot(HaveOccurred(), "Failed to generate the install time CRs and manifests")

			By("validating CRs and manifests were created")
			installCRsDir := fmt.Sprintf("%s/siteconfig/out/generated_installCRs/", siteConfigPath)
			siteDirs, err := os.ReadDir(installCRsDir)
			Expect(err).ToNot(HaveOccurred(), "Failed to read installed CRs directory: %s", installCRsDir)

			for _, dir := range siteDirs {
				siteDirPath := installCRsDir + dir.Name()
				files, err := os.ReadDir(siteDirPath)

				Expect(err).ToNot(HaveOccurred(), "Failed to read files in site directory %s", siteDirPath)
				Expect(len(files)).To(
					BeNumerically(">", 9), "Failed to generate at least 9 files in site directory %s", siteDirPath)
			}

			By("generating the policies")
			_, err = cluster.ExecLocalCommand(
				time.Minute,
				"podman",
				"run",
				"--rm",
				"-v",
				fmt.Sprintf("%s/policygentemplates/:/resources:Z", siteConfigPath),
				fmt.Sprintf("%s:%s", RANConfig.ZtpSiteGenerateImage, ztpImageTag),
				"generator",
				"config",
				".")
			Expect(err).ToNot(HaveOccurred(), "Failed to generate policies")

			By("validating the policies were created")
			expectedKind := []string{"Policy", "PlacementRule", "PlacementBinding"}

			// Expect to have at least 3 subdirs - common, group DU, site
			policyCRsDir := fmt.Sprintf("%s/policygentemplates/out/generated_configCRs/", siteConfigPath)
			configDirs, err := os.ReadDir(policyCRsDir)
			Expect(err).ToNot(HaveOccurred(), "Failed to list generated CRs directory")
			Expect(len(configDirs)).To(BeNumerically(">=", 3), "Not enough entries in generated CRs directory")

			for _, dir := range configDirs {
				dirPath := policyCRsDir + dir.Name()
				files, err := os.ReadDir(dirPath)
				Expect(err).ToNot(HaveOccurred(), "Failed to list files in %s", dirPath)
				Expect(len(files)).To(BeNumerically(">=", 3), "Not enough files in directory %s", dirPath)

				for _, file := range files {
					filePath := dirPath + "/" + file.Name()
					fileBytes, err := os.ReadFile(filePath)
					Expect(err).ToNot(HaveOccurred(), "Failed to read file %s", filePath)

					fileContent := make(map[string]interface{})
					err = yaml.Unmarshal(fileBytes, &fileContent)
					Expect(err).ToNot(HaveOccurred(), "Failed to unmarshal file %s as yaml", filePath)

					kind, ok := fileContent["kind"].(string)
					Expect(ok).To(BeTrue(), "Failed to cast file %s kind to string", filePath)
					Expect(kind).To(BeElementOf(expectedKind), "File %s is not one of the expected kinds", filePath)
				}
			}
		})
})
