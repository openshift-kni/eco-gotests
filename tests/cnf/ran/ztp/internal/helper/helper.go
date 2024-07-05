package helper

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift-kni/eco-goinfra/pkg/clients"
	"github.com/openshift-kni/eco-goinfra/pkg/ocm"
	"github.com/openshift-kni/eco-gotests/tests/cnf/ran/ztp/internal/tsparams"
	"k8s.io/apimachinery/pkg/util/wait"
)

// WaitForPolicyToExist waits for up to the specified timeout until the policy exists.
func WaitForPolicyToExist(
	client *clients.Settings, name, namespace string, timeout time.Duration) (*ocm.PolicyBuilder, error) {
	var policy *ocm.PolicyBuilder

	err := wait.PollUntilContextTimeout(
		context.TODO(), tsparams.ArgoCdChangeInterval, timeout, true, func(ctx context.Context) (bool, error) {
			var err error
			policy, err = ocm.PullPolicy(client, name, namespace)

			if err == nil {
				return true, nil
			}

			if strings.Contains(err.Error(), "does not exist") {
				return false, nil
			}

			return false, err
		})

	return policy, err
}

// WaitUntilSearchCollectorEnabled waits up to timeout until the KAC has the search collector addon enabled.
func WaitUntilSearchCollectorEnabled(kac *ocm.KACBuilder, timeout time.Duration) error {
	glog.V(tsparams.LogLevel).Infof(
		"Waiting until search collector is enabled for KAC %s in namespace %s", kac.Definition.Name, kac.Definition.Namespace)

	return wait.PollUntilContextTimeout(
		context.TODO(), tsparams.ArgoCdChangeInterval, timeout, true, func(ctx context.Context) (bool, error) {
			if !kac.Exists() {
				glog.V(tsparams.LogLevel).Infof(
					"KAC %s in namespace %s does not exist", kac.Definition.Name, kac.Definition.Namespace)

				return false, fmt.Errorf("kac %s in namespace %s does not exist", kac.Definition.Name, kac.Definition.Namespace)
			}

			return kac.Definition.Spec.SearchCollectorConfig.Enabled, nil
		})
}
