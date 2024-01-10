package ecore_system_test

import (
	"fmt"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-kni/eco-goinfra/pkg/storage"
	. "github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreinittools"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/ecore/internal/ecoreparams"
)

var _ = Describe(
	"ECore ODF Persistent Storage",
	Ordered,
	ContinueOnFailure,
	Label(ecoreparams.LabelEcoreValidateODFStorage), func() {
		Describe("StorageClasses", Label("ecore_sc_exist"), func() {
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
	})
