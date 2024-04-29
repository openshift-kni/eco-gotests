package helper

import (
	"errors"
	"fmt"

	"github.com/openshift-kni/eco-goinfra/pkg/pod"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/helper"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/tsparams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VerifyTalmIsInstalled checks that talm pod and container is present and that CGUs can be fetched.
func VerifyTalmIsInstalled() error {
	// Check for talm pods
	talmPods, err := pod.List(
		raninittools.APIClient,
		tsparams.OpenshiftOperatorNamespace,
		metav1.ListOptions{LabelSelector: tsparams.TalmPodLabelSelector})
	if err != nil {
		return err
	}

	// Check if any pods exist
	if len(talmPods) == 0 {
		return errors.New("unable to find talm pod")
	}

	// Check each pod for the talm container
	for _, talmPod := range talmPods {
		if !helper.IsPodHealthy(talmPod) {
			return fmt.Errorf("talm pod %s is not healthy", talmPod.Definition.Name)
		}

		if !helper.DoesContainerExistInPod(talmPod, tsparams.TalmContainerName) {
			return errors.New("talm pod defined but talm container does not exist")
		}
	}

	return nil
}
