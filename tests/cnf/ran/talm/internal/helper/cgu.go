package helper

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	. "github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/tsparams"
	configv1 "github.com/openshift/api/config/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
)

// WaitForCguInCondition waits up to timeout until the provided cguBuilder matches the expected status. Only the Type,
// Status, Reason, and Message fields of expected are checked.
func WaitForCguInCondition(
	cguBuilder *cgu.CguBuilder,
	expected metav1.Condition,
	timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		context.TODO(), 10*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			if !cguBuilder.Exists() {
				glog.V(tsparams.LogLevel).Infof(
					"cgu %s does not exist in namespace %s", cguBuilder.Definition.Name, cguBuilder.Definition.Namespace)

				return false, nil
			}

			for _, condition := range cguBuilder.Object.Status.Conditions {
				glog.V(tsparams.LogLevel).Infof("checking if condition %v matches the expected %v", condition, expected)

				matches := true

				if expected.Type != "" && condition.Type != expected.Type {
					matches = false
				}

				if matches && expected.Status != "" && condition.Status != expected.Status {
					matches = false
				}

				if matches && expected.Message != "" && !strings.Contains(condition.Message, expected.Message) {
					matches = false
				}

				if matches && expected.Reason != "" && condition.Reason != expected.Reason {
					matches = false
				}

				if matches {
					return true, nil
				}
			}

			return false, nil
		},
	)
}

// WaitForCguTimeout waits up to timeout until the provided cguBuilder matches the condition for a timeout.
func WaitForCguTimeout(cguBuilder *cgu.CguBuilder, timeout time.Duration) error {
	return WaitForCguInCondition(
		cguBuilder,
		metav1.Condition{
			Type:   tsparams.SucceededType,
			Reason: tsparams.TimedOutReason,
		},
		timeout)
}

// WaitForCguTimeoutMessage waits up to timeout until the provided cguBuilder matches the condition for a timeout.
func WaitForCguTimeoutMessage(cguBuilder *cgu.CguBuilder, timeout time.Duration) error {
	return WaitForCguInCondition(
		cguBuilder,
		metav1.Condition{
			Type:    tsparams.SucceededType,
			Message: tsparams.TalmTimeoutMessage,
		},
		timeout)
}

// WaitForCguTimeoutCanary waits up to timeout until the provided cguBuilder matches the condition for a timeout due to
// canary clusters.
func WaitForCguTimeoutCanary(cguBuilder *cgu.CguBuilder, timeout time.Duration) error {
	return WaitForCguInCondition(
		cguBuilder,
		metav1.Condition{
			Type:    tsparams.SucceededType,
			Message: tsparams.TalmCanaryTimeoutMessage,
		},
		timeout)
}

// WaitForCguSuccessfulFinish waits up to the timeout until the provided cguBuilder matches the condition for a
// successful finish.
func WaitForCguSuccessfulFinish(cguBuilder *cgu.CguBuilder, timeout time.Duration) error {
	return WaitForCguInCondition(
		cguBuilder,
		metav1.Condition{
			Type:   tsparams.SucceededType,
			Reason: tsparams.CompletedReason,
		},
		timeout)
}

// WaitForCguSucceeded waits for up to the timeout until the provided cguBuilder matches the condition for a success.
func WaitForCguSucceeded(cguBuilder *cgu.CguBuilder, timeout time.Duration) error {
	return WaitForCguInCondition(
		cguBuilder,
		metav1.Condition{
			Type:   tsparams.SucceededType,
			Status: metav1.ConditionTrue,
		},
		timeout)
}

// WaitForCguBlocked waits up to the timeout until the provided cguBuilder matches the condition for being blocked.
func WaitForCguBlocked(cguBuilder *cgu.CguBuilder, message string) error {
	return WaitForCguInCondition(
		cguBuilder,
		metav1.Condition{
			Type:    tsparams.ProgressingType,
			Status:  metav1.ConditionFalse,
			Message: message,
		},
		6*time.Minute)
}

// WaitForCguPreCacheValid waits up to the timeout until the provided cguBuilder matches the condition for valid
// precaching.
func WaitForCguPreCacheValid(cguBuilder *cgu.CguBuilder, timeout time.Duration) error {
	return WaitForCguInCondition(
		cguBuilder,
		metav1.Condition{
			Type:    tsparams.PreCacheValidType,
			Status:  metav1.ConditionTrue,
			Message: tsparams.PreCacheValidMessage,
		},
		timeout)
}

// WaitForCguPreCachePartiallyDone waits up to the timeout until the provided cguBuilder matches the condition for
// precaching being partially done.
func WaitForCguPreCachePartiallyDone(cguBuilder *cgu.CguBuilder, timeout time.Duration) error {
	return WaitForCguInCondition(
		cguBuilder,
		metav1.Condition{
			Type:    tsparams.PreCacheSucceededType,
			Status:  metav1.ConditionTrue,
			Message: tsparams.PreCachePartialFailMessage,
			Reason:  tsparams.PartiallyDoneReason,
		},
		timeout)
}

// IsClusterInCguInProgress checks if the current batch remediation progress for the provided cluster is InProgress.
func IsClusterInCguInProgress(cguBuilder *cgu.CguBuilder, cluster string) (bool, error) {
	if !cguBuilder.Exists() {
		return false, errors.New("provided CGU does not exist on client")
	}

	status, ok := cguBuilder.Object.Status.Status.CurrentBatchRemediationProgress[cluster]
	if !ok {
		glog.V(tsparams.LogLevel).Infof(
			"cluster %s not found in batch remediation progress for cgu %s in namespace %s",
			cluster, cguBuilder.Definition.Name, cguBuilder.Definition.Namespace)

		return false, nil
	}

	return status.State == "InProgress", nil
}

// isClusterInCguCompleted checks if the current batch remediation progress for the provided cluster is Completed.
func isClusterInCguCompleted(cguBuilder *cgu.CguBuilder, cluster string) (bool, error) {
	if !cguBuilder.Exists() {
		return false, errors.New("provided CGU does not exist on client")
	}

	status, ok := cguBuilder.Object.Status.Status.CurrentBatchRemediationProgress[cluster]
	if !ok {
		glog.V(tsparams.LogLevel).Infof(
			"cluster %s not found in batch remediation progress for cgu %s in namespace %s",
			cluster, cguBuilder.Definition.Name, cguBuilder.Definition.Namespace)

		return false, nil
	}

	return status.State == "Completed", nil
}

// WaitForClusterInCguInProgress waits up to timeout for the current batch remediation progress for the provided cluster
// to show InProgress.
func WaitForClusterInCguInProgress(cguBuilder *cgu.CguBuilder, cluster string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		context.TODO(), 15*time.Second, timeout, true, func(context.Context) (bool, error) {
			return IsClusterInCguInProgress(cguBuilder, cluster)
		})
}

// WaitForClusterInCguCompleted waits up to timeout for the current batch remediation progress for the provided cluster
// to show Completed.
func WaitForClusterInCguCompleted(cguBuilder *cgu.CguBuilder, cluster string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		context.TODO(), 15*time.Second, timeout, true, func(context.Context) (bool, error) {
			return isClusterInCguCompleted(cguBuilder, cluster)
		})
}

// SetupCguWithNamespace creates the policy with a namespace and its components for a cguBuilder then creates the
// cguBuilder.
func SetupCguWithNamespace(cguBuilder *cgu.CguBuilder, suffix string) (*cgu.CguBuilder, error) {
	// The client doesn't matter since we only want the definition. Kind and APIVersion are necessary for TALM.
	tempNs := namespace.NewBuilder(HubAPIClient, tsparams.TemporaryNamespace+suffix)
	tempNs.Definition.Kind = "Namespace"
	tempNs.Definition.APIVersion = corev1.SchemeGroupVersion.Version

	_, err := CreatePolicy(HubAPIClient, tempNs.Definition, suffix)
	if err != nil {
		return nil, err
	}

	err = CreatePolicyComponents(
		HubAPIClient, suffix, cguBuilder.Definition.Spec.Clusters, metav1.LabelSelector{})
	if err != nil {
		return nil, err
	}

	err = waitForPolicyComponentsExist(HubAPIClient, suffix)
	if err != nil {
		return nil, err
	}

	return cguBuilder.Create()
}

// SetupCguWithCatSrc creates the policy with a catalog source and its components for a cguBuilder then creates the
// cguBuilder.
func SetupCguWithCatSrc(cguBuilder *cgu.CguBuilder) (*cgu.CguBuilder, error) {
	// The client doesn't matter since we only want the definition. Kind and APIVersion are necessary for TALM.
	catsrc := olm.NewCatalogSourceBuilder(
		HubAPIClient, tsparams.CatalogSourceName, tsparams.TemporaryNamespace)
	catsrc.Definition.Spec.SourceType = operatorsv1alpha1.SourceTypeInternal
	catsrc.Definition.Spec.Priority = 1
	catsrc.Definition.Kind = "CatalogSource"
	catsrc.Definition.APIVersion = "operators.coreos.com/v1alpha1"

	_, err := CreatePolicy(HubAPIClient, catsrc.Definition, "")
	if err != nil {
		return nil, err
	}

	err = CreatePolicyComponents(
		HubAPIClient, "", cguBuilder.Definition.Spec.Clusters, metav1.LabelSelector{})
	if err != nil {
		return nil, err
	}

	err = waitForPolicyComponentsExist(HubAPIClient, "")
	if err != nil {
		return nil, err
	}

	return cguBuilder.Create()
}

// WaitToEnableCgu waits for the TalmSystemStablizationTime before enabling the CGU.
func WaitToEnableCgu(cguBuilder *cgu.CguBuilder) (*cgu.CguBuilder, error) {
	time.Sleep(tsparams.TalmSystemStablizationTime)

	cguBuilder.Definition.Spec.Enable = ptr.To(true)

	return cguBuilder.Update(true)
}

// SetupCguWithClusterVersion creates the policy with the provided clustrer version and its components for a cguBuilder
// then creates the cguBuilder.
func SetupCguWithClusterVersion(
	cguBuilder *cgu.CguBuilder, clusterVersion *configv1.ClusterVersion) (*cgu.CguBuilder, error) {
	_, err := CreatePolicy(HubAPIClient, clusterVersion, "")
	if err != nil {
		return nil, err
	}

	err = CreatePolicyComponents(
		HubAPIClient, "", cguBuilder.Definition.Spec.Clusters, metav1.LabelSelector{})
	if err != nil {
		return nil, err
	}

	err = waitForPolicyComponentsExist(HubAPIClient, "")
	if err != nil {
		return nil, err
	}

	return cguBuilder.Create()
}

// waitForPolicyComponentsExist waits until all the policy components exist on the provided client.
func waitForPolicyComponentsExist(client *clients.Settings, suffix string) error {
	return wait.PollUntilContextTimeout(
		context.TODO(), 15*time.Second, 5*time.Minute, false, func(ctx context.Context) (bool, error) {
			// Check for definitions being nil too since sometimes the err is nil when the resource does not
			// exist on the cluster.
			policy, err := ocm.PullPolicy(client, tsparams.PolicyName+suffix, tsparams.TestNamespace)
			if err != nil || policy.Definition == nil {
				return false, nil
			}

			policySet, err := ocm.PullPolicySet(client, tsparams.PolicySetName+suffix, tsparams.TestNamespace)
			if err != nil || policySet.Definition == nil {
				return false, nil
			}

			placementRule, err := ocm.PullPlacementRule(client, tsparams.PlacementRuleName+suffix, tsparams.TestNamespace)
			if err != nil || placementRule.Definition == nil {
				return false, nil
			}

			placementBinding, err := ocm.PullPlacementBinding(
				client, tsparams.PlacementBindingName+suffix, tsparams.TestNamespace)
			if err != nil || placementBinding.Definition == nil {
				return false, nil
			}

			return true, nil
		})
}
