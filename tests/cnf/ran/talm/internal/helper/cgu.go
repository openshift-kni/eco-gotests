package helper

import (
	"context"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/cgu"
	"github.com/openshift-kni/eco-goinfra/pkg/namespace"
	"github.com/openshift-kni/eco-goinfra/pkg/olm"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/talm/internal/tsparams"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
	conditionType := tsparams.SucceededType
	conditionReason := "TimedOut"

	return WaitForCguInCondition(cguBuilder, metav1.Condition{Type: conditionType, Reason: conditionReason}, timeout)
}

// WaitForCguTimeoutMessage waits up to timeout until the provided cguBuilder matches the condition for a timeout.
func WaitForCguTimeoutMessage(cguBuilder *cgu.CguBuilder, timeout time.Duration) error {
	conditionType := tsparams.SucceededType
	conditionMessage := tsparams.TalmTimeoutMessage

	return WaitForCguInCondition(cguBuilder, metav1.Condition{Type: conditionType, Message: conditionMessage}, timeout)
}

// WaitForCguTimeoutCanary waits up to timeout until the provided cguBuilder matches the condition for a timeout due to
// canary clusters.
func WaitForCguTimeoutCanary(cguBuilder *cgu.CguBuilder, timeout time.Duration) error {
	conditionType := tsparams.SucceededType
	conditionMessage := "Policy remediation took too long on canary clusters"

	return WaitForCguInCondition(cguBuilder, metav1.Condition{Type: conditionType, Message: conditionMessage}, timeout)
}

// WaitForCguSuccessfulFinish waits up to the timeout until the provided cguBuilder matches the condition for a
// successful finish.
func WaitForCguSuccessfulFinish(cguBuilder *cgu.CguBuilder, timeout time.Duration) error {
	conditionType := tsparams.SucceededType
	conditionReason := tsparams.CompletedReason

	return WaitForCguInCondition(cguBuilder, metav1.Condition{Type: conditionType, Reason: conditionReason}, timeout)
}

// WaitForCguSucceeded waits for up to the timeout until the provided cguBuilder matches the condition for a success.
func WaitForCguSucceeded(cguBuilder *cgu.CguBuilder, timeout time.Duration) error {
	conditionType := tsparams.SucceededType
	conditionStatus := metav1.ConditionTrue

	return WaitForCguInCondition(cguBuilder, metav1.Condition{Type: conditionType, Status: conditionStatus}, timeout)
}

// SetupCguWithNamespace creates the policy with a namespace and its components for a cguBuilder then creates the
// cguBuilder.
func SetupCguWithNamespace(cguBuilder *cgu.CguBuilder, suffix string) (*cgu.CguBuilder, error) {
	// The client doesn't matter since we only want the definition. Kind and APIVersion are necessary for TALM.
	tempNs := namespace.NewBuilder(raninittools.HubAPIClient, tsparams.TemporaryNamespace+suffix)
	tempNs.Definition.Kind = "Namespace"
	tempNs.Definition.APIVersion = corev1.SchemeGroupVersion.Version

	_, err := CreatePolicy(raninittools.HubAPIClient, tempNs.Definition, suffix)
	if err != nil {
		return nil, err
	}

	err = CreatePolicyComponents(
		raninittools.HubAPIClient, suffix, cguBuilder.Definition.Spec.Clusters, metav1.LabelSelector{})
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
		raninittools.HubAPIClient, tsparams.CatalogSourceName, tsparams.TemporaryNamespace)
	catsrc.Definition.Spec.SourceType = operatorsv1alpha1.SourceTypeInternal
	catsrc.Definition.Spec.Priority = 1
	catsrc.Definition.Kind = "CatalogSource"
	catsrc.Definition.APIVersion = "operators.coreos.com/v1alpha1"

	_, err := CreatePolicy(raninittools.HubAPIClient, catsrc.Definition, "")
	if err != nil {
		return nil, err
	}

	err = CreatePolicyComponents(
		raninittools.HubAPIClient, "", cguBuilder.Definition.Spec.Clusters, metav1.LabelSelector{})
	if err != nil {
		return nil, err
	}

	return cguBuilder.Create()
}
