package tests

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-kni/eco-goinfra/pkg/configmap"
	"github.com/openshift-kni/eco-goinfra/pkg/kmm"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/define"
	"github.com/openshift-kni/eco-gotests/tests/hw-accel/kmm/modules/internal/tsparams"
	. "github.com/openshift-kni/eco-gotests/tests/internal/inittools"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("KMM", Ordered, Label(tsparams.LabelSuite, tsparams.LabelSanity), func() {

	Context("Operator images", Label("clamav"), func() {
		relatedImgMap := map[string]string{}
		scanner := "scanner"
		image := fmt.Sprintf("%s/%s/%s:latest",
			tsparams.LocalImageRegistry, tsparams.ScannerTestNamespace, scanner)
		var containerImg []string
		var dockerfileConfigMap *configmap.Builder

		BeforeAll(func() {
			By("Create Namespace")
			_, err := namespace.NewBuilder(APIClient, tsparams.ScannerTestNamespace).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating namespace")

			By("Create Configmap")
			configmMapContents := define.KmmScannerConfigMapContents()
			dockerfileConfigMap, err = configmap.NewBuilder(APIClient, scanner, tsparams.ScannerTestNamespace).
				WithData(configmMapContents).Create()
			Expect(err).ToNot(HaveOccurred(), "error creating configmap")
		})

		AfterAll(func() {
			By("Delete Module")
			_, err := kmm.NewModuleBuilder(APIClient, scanner, tsparams.ScannerTestNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting scanner module")

			By("Await module to be deleted")
			err = await.ModuleObjectDeleted(APIClient, scanner, tsparams.ScannerTestNamespace, time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while waiting module to be deleted")

			By("Delete namespace")
			err = namespace.NewBuilder(APIClient, tsparams.ScannerTestNamespace).Delete()
			Expect(err).ToNot(HaveOccurred(), "error deleting namespace")
		})

		It("should pass malware testing", polarion.ID("68147"), func() {
			By("Obtain KMM images for test")
			pods, _ := pod.List(APIClient, "openshift-kmm", v1.ListOptions{
				FieldSelector: "status.phase=Running",
			})

			for _, pod := range pods {
				for _, container := range pod.Object.Spec.Containers {
					containerImg = append(containerImg, container.Image)
					for _, env := range container.Env {
						if env.Name == tsparams.RelImgSign {
							relatedImgMap[tsparams.RelImgSign] = env.Value
						}
						if env.Name == tsparams.RelImgWorker {
							relatedImgMap[tsparams.RelImgWorker] = env.Value
						}
						if env.Name == tsparams.RelImgMustGather {
							relatedImgMap[tsparams.RelImgMustGather] = env.Value
						}
					}
				}
			}

			glog.V(kmmparams.KmmLogLevel).Infof("related: %s", relatedImgMap)
			glog.V(kmmparams.KmmLogLevel).Infof("container: %s", containerImg)

			By("Create kernel mapping")
			kernelMapping := kmm.NewRegExKernelMappingBuilder("^.+$")

			kernelMapping.WithContainerImage(image).
				WithBuildArg("OPERATOR_IMAGE", containerImg[0]).
				WithBuildArg("RBAC_IMAGE", containerImg[1]).
				WithBuildArg(tsparams.RelImgSign, relatedImgMap[tsparams.RelImgSign]).
				WithBuildArg(tsparams.RelImgWorker, relatedImgMap[tsparams.RelImgWorker]).
				WithBuildArg(tsparams.RelImgMustGather, relatedImgMap[tsparams.RelImgMustGather]).
				WithBuildDockerCfgFile(dockerfileConfigMap.Object.Name)
			kerMalOne, err := kernelMapping.BuildKernelMappingConfig()
			Expect(err).ToNot(HaveOccurred(), "error creating kernel mapping")

			By("Create moduleloader")
			moduleLoaderContainer := kmm.NewModLoaderContainerBuilder(scanner)
			moduleLoaderContainer.WithKernelMapping(kerMalOne)
			moduleLoaderContainer.WithVersion("never")
			moduleLoaderContainerCfg, err := moduleLoaderContainer.BuildModuleLoaderContainerCfg()
			Expect(err).ToNot(HaveOccurred(), "Error creating moduleloadercontainer")

			By("Create module")
			module := kmm.NewModuleBuilder(APIClient, scanner, tsparams.ScannerTestNamespace).
				WithNodeSelector(GeneralConfig.WorkerLabelMap)
			module = module.WithModuleLoaderContainer(moduleLoaderContainerCfg)
			_, err = module.Create()
			Expect(err).ToNot(HaveOccurred(), "error creating scanner")

			By("Await build pod to complete build")
			err = await.BuildPodCompleted(APIClient, tsparams.ScannerTestNamespace, 10*time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error while buildging scanner")

			By("Run pod from the scanner image")
			scannerPod := pod.NewBuilder(APIClient, "scan-checker", tsparams.ScannerTestNamespace, image)
			_, err = scannerPod.CreateAndWaitUntilRunning(time.Minute)
			Expect(err).ToNot(HaveOccurred(), "error running scanner image")

			By("Checking operator image scan")
			buff, err := scannerPod.ExecCommand([]string{"tail", "-11", "operator.log"})
			Expect(err).ToNot(HaveOccurred(), "error verifying operator log")
			opContents := buff.String()
			glog.V(kmmparams.KmmLogLevel).Infof(opContents)
			Expect(strings.Contains(opContents, "Infected files: 0")).To(BeTrue(), "infected files not zero")

			By("Checking must-gather image scan")
			buff, err = scannerPod.ExecCommand([]string{"tail", "-11", "must-gather.log"})
			Expect(err).ToNot(HaveOccurred(), "error verifying must-gather log")
			opContents = buff.String()
			glog.V(kmmparams.KmmLogLevel).Infof(opContents)
			Expect(strings.Contains(opContents, "Infected files: 0")).To(BeTrue(), "infected files not zero")

			By("Checking sign image scan")
			buff, err = scannerPod.ExecCommand([]string{"tail", "-11", "sign.log"})
			Expect(err).ToNot(HaveOccurred(), "error verifying sign log")
			opContents = buff.String()
			glog.V(kmmparams.KmmLogLevel).Infof(opContents)
			Expect(strings.Contains(opContents, "Infected files: 0")).To(BeTrue(), "infected files not zero")

			By("Checking worker image scan")
			buff, err = scannerPod.ExecCommand([]string{"tail", "-11", "worker.log"})
			Expect(err).ToNot(HaveOccurred(), "error verifying worker log")
			opContents = buff.String()
			glog.V(kmmparams.KmmLogLevel).Infof(opContents)
			Expect(strings.Contains(opContents, "Infected files: 0")).To(BeTrue(), "infected files not zero")

			By("Checking rbac image scan")
			buff, err = scannerPod.ExecCommand([]string{"tail", "-11", "rbac.log"})
			Expect(err).ToNot(HaveOccurred(), "error verifying rbac log")
			opContents = buff.String()
			glog.V(kmmparams.KmmLogLevel).Infof(opContents)
			Expect(strings.Contains(opContents, "Infected files: 0")).To(BeTrue(), "infected files not zero")

		})
	})
})
