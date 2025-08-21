package await

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/olm"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/hw-accel/kmm/internal/kmmparams"
	"k8s.io/apimachinery/pkg/util/wait"
)

// OperatorUpgrade awaits operator upgrade to semver version.
func OperatorUpgrade(apiClient *clients.Settings, semver string, timeout time.Duration) error {
	return wait.PollUntilContextTimeout(
		context.TODO(), time.Second, timeout, true, func(ctx context.Context) (bool, error) {
			csv, err := olm.ListClusterServiceVersionWithNamePattern(apiClient, "kernel",
				kmmparams.KmmOperatorNamespace)

			for _, c := range csv {
				glog.V(kmmparams.KmmLogLevel).Infof("CSV: %s, Version: %s, Status: %s",
					c.Object.Spec.DisplayName, c.Object.Spec.Version, c.Object.Status.Phase)
			}

			for _, c := range csv {
				return c.Object.Spec.Version.String() == semver && c.Object.Status.Phase == "Succeeded", nil
			}

			return false, err
		})
}
