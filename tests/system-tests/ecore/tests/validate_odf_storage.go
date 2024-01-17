package ecore_system_test

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/storage"
	"github.com/openshift-kni/eco-gotests/tests/internal/polarion"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore ODF Persistent Storage",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidateODFStorage), func() {
		Describe("StorageClasses", Label("ecore_sc_exist"), polarion.ID("62960"), func() {
			It("Asserts storageClasses exist", func() {
				glog.V(ecoreparams.ECoreLogLevel).Infof("Validating StorageClasses")
				for sc, provisioner := range ECoreConfig.StorageClassesMap {
					eClass := storage.NewClassBuilder(APIClient, sc, provisioner)
					glog.V(ecoreparams.ECoreLogLevel).Infof("Assert storageClass %q exists", sc)
					scExists := eClass.Exists()
					Expect(scExists).To(BeTrue(), fmt.Sprintf("StorageClass %q not found", sc))
				}
			})
		}) // StorageClasses exist

		Context("Create PVC based on StorageClass", Label("ecore_odf_pvc_per_sc"), func() {

			var testNSName = "qe-odf-ns"

			BeforeAll(func() {
				By("Asserting test namespace already exists")
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Assert if namespace %q exists", testNSName))

				if prevNS, err := namespace.Pull(APIClient, testNSName); err == nil {
					glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Namespace %q exists. Removing...", testNSName))
					delErr := prevNS.DeleteAndWait(6 * time.Minute)
					Expect(delErr).ToNot(HaveOccurred(), fmt.Sprintf("Failed to delete %q namespace", testNSName))
				}

				By("Creating a test namespace")
				testNS := namespace.NewBuilder(APIClient, testNSName)
				testNS, err := testNS.Create()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create test namespace %s", testNSName))
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("Namespace %q created", testNS.Object.Name))
			})

			It("Asserts CephFS PVC creation", func(ctx SpecContext) {
				By("Creating new PVC Builder")
				myPVC := storage.NewPVCBuilder(APIClient, "telco-cephfs-pvc", testNSName)
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("PVC\n%#v", myPVC))

				By("Setting AccessMode")
				myPVC, err := myPVC.WithPVCAccessMode("ReadWriteOnce")
				Expect(err).ToNot(HaveOccurred(), "Failed to set AccessMode")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC accessMode: ", myPVC.Definition.Spec.AccessModes)

				By("Setting PVC capacity")
				myPVC, err = myPVC.WithPVCCapacity("5G")
				Expect(err).ToNot(HaveOccurred(), "Failed to set Capacity")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC Capacity: ", myPVC.Definition.Spec.Resources)

				By("Setting StorageClass for PVC")
				myPVC, err = myPVC.WithStorageClass("ocs-external-storagecluster-cephfs")
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to use StorageClass: %v", err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC StorageClass: ", myPVC.Definition.Spec.StorageClassName)

				By("Creating PVC")
				myPVC, err = myPVC.Create()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create PVC: %v", err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("Created PVC: ", myPVC.Object.Status)

				Eventually(func(phase string) bool {
					if ok := myPVC.Exists(); ok {
						glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("\tPVC Phase is %q", myPVC.Object.Status.Phase))

						return string(myPVC.Object.Status.Phase) == phase
					}

					return false
				}, 5*time.Minute, 3*time.Second).WithContext(ctx).WithArguments("Bound").Should(BeTrue(),
					fmt.Sprintf("Unexpeced PVC state %q", myPVC.Object.Status.Phase))
			})

			It("Asserts Ceph RBD PVC creation", func(ctx SpecContext) {
				By("Creating new PVC Builder")
				myPVC := storage.NewPVCBuilder(APIClient, "telco-cephrbd-pvc", testNSName)
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("PVC\n%#v", myPVC))

				By("Setting AccessMode")
				myPVC, err := myPVC.WithPVCAccessMode("ReadWriteOnce")
				Expect(err).ToNot(HaveOccurred(), "Failed to set AccessMode")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC accessMode: ", myPVC.Definition.Spec.AccessModes)

				By("Setting PVC capacity")
				myPVC, err = myPVC.WithPVCCapacity("5G")
				Expect(err).ToNot(HaveOccurred(), "Failed to set Capacity")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC Capacity: ", myPVC.Definition.Spec.Resources)

				By("Setting StorageClass for PVC")
				myPVC, err = myPVC.WithStorageClass("ocs-external-storagecluster-ceph-rbd")
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to use StorageClass: %v", err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC StorageClass: ", myPVC.Definition.Spec.StorageClassName)

				By("Creating PVC")
				myPVC, err = myPVC.Create()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create PVC: %v", err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("Created PVC: ", myPVC.Object.Status)

				Eventually(func(phase string) bool {
					if ok := myPVC.Exists(); ok {
						glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("\tPVC Phase is %q", myPVC.Object.Status.Phase))

						return string(myPVC.Object.Status.Phase) == phase
					}

					return false
				}, 5*time.Minute, 3*time.Second).WithContext(ctx).WithArguments("Bound").Should(BeTrue(),
					fmt.Sprintf("Unexpeced PVC state %q", myPVC.Object.Status.Phase))
			})

			It("Asserts Ceph Nooba PVC creation", func(ctx SpecContext) {
				By("Creating new PVC Builder")
				myPVC := storage.NewPVCBuilder(APIClient, "telco-nooba-pvc", testNSName)
				glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("PVC\n%#v", myPVC))

				By("Setting AccessMode")
				myPVC, err := myPVC.WithPVCAccessMode("ReadWriteOnce")
				Expect(err).ToNot(HaveOccurred(), "Failed to set AccessMode")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC accessMode: ", myPVC.Definition.Spec.AccessModes)

				By("Setting PVC capacity")
				myPVC, err = myPVC.WithPVCCapacity("5G")
				Expect(err).ToNot(HaveOccurred(), "Failed to set Capacity")
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC Capacity: ", myPVC.Definition.Spec.Resources)

				By("Setting StorageClass for PVC")
				myPVC, err = myPVC.WithStorageClass("openshift-storage.noobaa.io")
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to use StorageClass: %v", err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("PVC StorageClass: ", myPVC.Definition.Spec.StorageClassName)

				By("Creating PVC")
				myPVC, err = myPVC.Create()
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf("Failed to create PVC: %v", err))
				glog.V(ecoreparams.ECoreLogLevel).Infof("Created PVC: ", myPVC.Object.Status)

				Eventually(func(phase string) bool {
					if ok := myPVC.Exists(); ok {
						glog.V(ecoreparams.ECoreLogLevel).Infof(fmt.Sprintf("\tPVC Phase is %q", myPVC.Object.Status.Phase))

						return string(myPVC.Object.Status.Phase) == phase
					}

					return false
				}, 5*time.Minute, 3*time.Second).WithContext(ctx).WithArguments("Pending").Should(BeTrue(),
					fmt.Sprintf("Unexpeced PVC state %q", myPVC.Object.Status.Phase))
			})
		}) // end Context
	})
