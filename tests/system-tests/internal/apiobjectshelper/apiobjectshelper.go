package apiobjectshelper

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/await"
	"github.com/openshift-kni/eco-gotests/tests/system-tests/internal/csv"
	"k8s.io/apimachinery/pkg/util/wait"
)

// VerifyNamespaceExists asserts specific namespace exists.
func VerifyNamespaceExists(apiClient *clients.Settings, nsname string, timeout time.Duration) error {
	glog.V(90).Infof("Verify namespace %q exists", nsname)

	err := wait.PollUntilContextTimeout(context.TODO(), time.Second, timeout, true,
		func(ctx context.Context) (bool, error) {
			_, pullErr := namespace.Pull(apiClient, nsname)
			if pullErr != nil {
				glog.V(90).Infof("Failed to pull in namespace %q - %v", nsname, pullErr)

				return false, nil
			}

			return true, nil
		})

	if err != nil {
		return fmt.Errorf("failed to pull in %s namespace", nsname)
	}

	return nil
}

// VerifyOperatorDeployment assert that specific deployment succeeded.
func VerifyOperatorDeployment(apiClient *clients.Settings,
	subscriptionName, deploymentName, nsname string, timeout time.Duration) error {
	glog.V(90).Infof("Verify deployment %s in namespace %s", deploymentName, nsname)

	csvName, err := csv.GetCurrentCSVNameFromSubscription(apiClient, subscriptionName, nsname)

	if err != nil {
		return fmt.Errorf("csv %s not found in namespace %s", csvName, nsname)
	}

	csvObj, err := olm.PullClusterServiceVersion(apiClient, csvName, nsname)

	if err != nil {
		return fmt.Errorf("failed to pull %q csv from the %s namespace", csvName, nsname)
	}

	isSuccessful, err := csvObj.IsSuccessful()

	if err != nil {
		return fmt.Errorf("failed to verify csv %s in the namespace %s status", csvName, nsname)
	}

	if !isSuccessful {
		return fmt.Errorf("failed to deploy %s; the csv %s in the namespace %s status %v",
			subscriptionName, csvName, nsname, isSuccessful)
	}

	glog.V(90).Infof("Confirm that operator %s is running in namespace %s", deploymentName, nsname)

	err = await.WaitUntilDeploymentReady(apiClient, deploymentName, nsname, timeout)

	if err != nil {
		return fmt.Errorf("deployment %s not found in %s namespace; %w", deploymentName, nsname, err)
	}

	return nil
}
