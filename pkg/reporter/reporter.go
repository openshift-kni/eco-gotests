package reporter

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/openshift-kni/eco-gotests/pkg/clients"
	"github.com/openshift-kni/k8sreporter"
	"k8s.io/apimachinery/pkg/runtime"
)

// Dump runs reports and collect logs under given directory.
func Dump(client *clients.Settings, crds []k8sreporter.CRData, namespace, reportDirPath, testSuiteName string) {
	dumpNamespace := func(ns string) bool {
		return strings.HasPrefix(ns, namespace)
	}
	//nolint:staticcheck
	addToScheme := func(scheme *runtime.Scheme) error {
		//nolint:ineffassign,staticcheck
		scheme = client.Client.Scheme()

		return nil
	}

	reporter, err := k8sreporter.New(
		client.KubeconfigPath, addToScheme, dumpNamespace, reportDirPath, crds...)

	if err != nil {
		log.Fatalf("Failed to initialize the reporter %s", err)
	}

	if err := os.MkdirAll(reportDirPath, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	reporter.Dump(10*time.Minute, testSuiteName)
}
